package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"runtime"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/wiggin77/cfg"
)

// Default config file directory and name.
const (
	appVersion        = "Dynip v1.0.0"
	defaultConfigDir  = ".config/dynip"
	defaultConfigFile = "dynip.conf"
)

type appResult struct {
	exitCode int
	exitMsg  string
}

// Application entry point
func main() {
	result := &appResult{}
	defer exit(result)

	var fileConfig string
	var daemon bool
	var verbose bool
	var install bool
	var uninstall bool
	var version bool
	var help bool
	defFileConfig := defConfigFile()

	// process command line flags
	flag.StringVar(&fileConfig, "f", defFileConfig, "config file")
	flag.BoolVar(&daemon, "d", false, "run as a daemon")
	flag.BoolVar(&verbose, "v", false, "overrides verbose setting in config (interactive only)")
	flag.BoolVar(&install, "i", false, "install as service/daemon")
	flag.BoolVar(&uninstall, "u", false, "uninstall service/daemon")
	flag.BoolVar(&version, "version", false, "display version info")
	flag.BoolVar(&help, "h", false, "display help")
	flag.Parse()

	// possibly display help
	if help {
		flag.PrintDefaults()
		return
	}

	// possibly display version info
	if version {
		fmt.Println(appVersion)
		return
	}

	// possibly install as service
	if install {
		err := serviceInstall()
		if err != nil {
			result.exitCode = -5
			result.exitMsg = fmt.Sprintf("%v", err)
		} else {
			result.exitMsg = "install successful"
		}
		return
	}

	// possibly uninstall
	if uninstall {
		err := serviceUninstall()
		if err != nil {
			result.exitCode = -6
			result.exitMsg = fmt.Sprintf("%v", err)
		} else {
			result.exitMsg = "uninstall successful"
		}
		return
	}

	// possibly run as a daemon
	if daemon {
		err := serviceRun()
		if err != nil {
			result.exitCode = -1
			result.exitMsg = fmt.Sprintf("%v", err)
		}
		return
	}

	err := runOnce(fileConfig, verbose)
	if err != nil {
		result.exitCode = -1
		result.exitMsg = fmt.Sprintf("%v", err)
	}
}

func runOnce(fileConfig string, verbose bool) error {
	// load config file
	appConfig, err := NewAppConfig(fileConfig)
	if err != nil {
		return err
	}

	// if verbose specified on command line it overrides config
	if verbose {
		src := cfg.NewSrcMapFromMap(map[string]string{"verbose": "YES"})
		appConfig.PrependSource(src)
	}

	// configure logger
	logger, err := configureLogging(appConfig)
	if err != nil {
		return err
	}
	c, ok := logger.Out.(io.Closer)
	if ok {
		defer func() { _ = c.Close() }()
	}
	_, err = updateIP(appConfig, logger)
	return err
}

func configureLogging(cfg *AppConfig) (*logrus.Logger, error) {
	file := cfg.getKeyVal(keyLogFile)
	syslogger := isTrue(cfg.getKeyVal(keySyslog))
	verbose := isTrue(cfg.getKeyVal(keyVerbose))

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
			hook, err := syslogHook("dynip")
			if err != nil {
				return nil, err
			}
			if hook != nil {
				logger.AddHook(hook)
			}
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
	var p string
	me, err := user.Current()
	if err == nil {
		p = me.HomeDir
	}
	return p, err
}

// Best guess determination for syslog support.
func hasSysLog() bool {
	return runtime.GOOS != "windows" && runtime.GOOS != "nacl" && runtime.GOOS != "plan9"
}

// Exit app with return code and optional error message.
func exit(result *appResult) {

	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "Panic: %s\n%s", r, debug.Stack())
		if result.exitCode == 0 {
			result.exitCode = -1
		}
	}
	if len(result.exitMsg) > 0 {
		out := os.Stdout
		if result.exitCode != 0 {
			out = os.Stderr
		}
		fmt.Fprintf(out, "%s\n", result.exitMsg)
	}
	os.Exit(result.exitCode)
}
