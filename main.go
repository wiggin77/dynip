package main

import (
	"flag"
	"fmt"
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

// Application entry point
func main() {
	var logger = log.New()
	logger.Level = log.InfoLevel
	logger.Out = os.Stdout

	var exitCode int
	var exitMsg string
	defer exit(&exitCode, &exitMsg, logger)

	var fileConfig string
	var fileLog string
	var verbose bool
	var daemon bool
	var syslogger bool
	defFileConfig := defConfigFile()

	// Process command line flags
	flag.StringVar(&fileConfig, "f", defFileConfig, "config file")
	flag.StringVar(&fileLog, "l", "", "optional log file for output")
	flag.BoolVar(&verbose, "v", false, "verbose output")
	flag.BoolVar(&daemon, "d", false, "run as a daemon")
	flag.BoolVar(&syslogger, "s", false, "log to syslog (Linux/*BSD/Mac only)")
	flag.Parse()

	// configure logging
	if fileLog != "" {
		if f, err := os.Create(fileLog); err != nil {
			exitCode = -10
			exitMsg = fmt.Sprintf("%v", err)
		} else {
			defer f.Close()
			logger.Out = f
		}
	}
	if syslogger && hasSysLog() {
		logger.Out = ioutil.Discard
		if hasSysLog() {
			hook, err := logrus_syslog.NewSyslogHook("", "", syslog.LOG_INFO, "dynip")
			if err != nil {
				exitCode = -20
				exitMsg = "Cannot connect to syslog daemon"
			}
			logger.AddHook(hook)
		}
	}
	if verbose {
		logger.Level = log.DebugLevel
	}

	// Load config file
	appConfig, err := NewAppConfig(fileConfig)
	if err != nil {
		exitCode = -7
		exitMsg = fmt.Sprintf("%v", err)
	}

	var rerr error
	if daemon {
		exit := make(chan string)
		rerr = runDaemon(appConfig, logger, exit)
	} else {
		rerr = updateIP(appConfig, logger)
	}
	if rerr != nil {
		exitCode = -1
		exitMsg = fmt.Sprintf("%v", rerr)
	}
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
func exit(code *int, msg *string, logger *logrus.Logger) {
	if r := recover(); r != nil {
		logger.Panicf("Panic: %s\n%s", r, debug.Stack())
		if *code == 0 {
			*code = -1
		}
	}
	if len(*msg) > 0 {
		if *code == 0 {
			logger.Info(*msg)
		} else {
			logger.Error(*msg)
		}
	}
	os.Exit(*code)
}
