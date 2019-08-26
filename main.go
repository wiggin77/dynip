package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log/syslog"
	"os"
	"os/user"
	"path"
	"runtime"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	logrus_syslog "github.com/sirupsen/logrus/hooks/syslog"
)

// Default config file directory and name.
const (
	defaultConfigDir  = ".config/dynip"
	defaultConfigFile = "dynip.conf"
)

type appResult struct {
	exitCode int
	exitMsg  string
}

type appLogger struct {
	logrus.Logger
	io.Closer
}

// Application entry point
func main() {
	result := &appResult{}
	var logger *logrus.Logger
	defer exit(result, logger)

	var fileConfig string
	var daemon bool
	var install bool
	defFileConfig := defConfigFile()

	// process command line flags
	flag.StringVar(&fileConfig, "f", defFileConfig, "config file")
	flag.BoolVar(&daemon, "d", false, "run as a daemon")
	flag.BoolVar(&install, "i", false, "install as service/daemon")
	flag.Parse()

	// possibly install as service
	if install {
		err := serviceInstall()
		if err != nil {
			result.exitCode = -5
			result.exitMsg = fmt.Sprintf("%v", err)
		}
		return
	}

	// load config file
	appConfig, err := NewAppConfig(fileConfig)
	if err != nil {
		result.exitCode = -10
		result.exitMsg = fmt.Sprintf("%v", err)
		return
	}

	// configure logger
	logger, err = configureLogging(appConfig)
	if err != nil {
		result.exitCode = -20
		result.exitMsg = fmt.Sprintf("%v", err)
		return
	}
	c, ok := logger.Out.(io.Closer)
	if ok {
		defer c.Close()
	}

	// run once, or continuously as daemon
	var rerr error
	if daemon {
		exit := make(chan string)
		runDaemon(appConfig, logger, exit)
	} else {
		_, rerr = updateIP(appConfig, logger)
	}
	if rerr != nil {
		result.exitCode = -1
		result.exitMsg = fmt.Sprintf("%v", rerr)
	}
}

func configureLogging(cfg *AppConfig) (*logrus.Logger, error) {
	file := cfg.getKeyVal(keyLogFile)
	syslogger := isTrue(cfg.getKeyVal(keySyslog))
	verbose := isTrue(cfg.getKeyVal(keyDebug))

	logger := log.New()
	logger.Level = log.InfoLevel
	logger.Out = os.Stdout

	if file != "" {
		f, err := os.Create(file)
		if err != nil {
			return nil, err
		}
		logger.Out = f
	}
	if syslogger {
		logger.Out = ioutil.Discard
		if hasSysLog() {
			hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "dynip")
			if err != nil {
				return nil, err
			}
			logger.AddHook(hook)
		} else {
			return nil, fmt.Errorf("syslog not supported for %s", runtime.GOOS)
		}
	}
	if verbose {
		logger.Level = log.DebugLevel
	}
	return logger, nil
}

// Get the filespec for the default config file in user's home directory or /etc.
func defConfigFile() string {
	home, err := homePath()
	if err == nil {
		home = path.Join(home, defaultConfigDir, defaultConfigFile)
	}
	if _, err := os.Stat(home); os.IsNotExist(err) {
		// Only use /etc if file exists there.
		etc := path.Join("/etc", defaultConfigFile)
		if _, err := os.Stat(etc); !os.IsNotExist(err) {
			home = etc
		}
	}
	return home
}

// Get user's home directory.
func homePath() (string, error) {
	var path string
	me, err := user.Current()
	if err == nil {
		path = me.HomeDir
	}
	return path, err
}

// Best guess determination for syslog support.
func hasSysLog() bool {
	return runtime.GOOS != "windows" && runtime.GOOS != "nacl" && runtime.GOOS != "plan9"
}

// Exit app with return code and optional error message.
func exit(result *appResult, logger *logrus.Logger) {
	if r := recover(); r != nil {
		logger.Panicf("Panic: %s\n%s", r, debug.Stack())
		if result.exitCode == 0 {
			result.exitCode = -1
		}
	}
	if len(result.exitMsg) > 0 {
		if result.exitCode == 0 {
			logger.Info(result.exitMsg)
		} else {
			logger.Error(result.exitMsg)
		}
	}
	os.Exit(result.exitCode)
}
