package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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
	appVersion        = "Dynip v1.1.0"
	defaultConfigDir  = ".config/dynip"
	defaultConfigFile = "dynip.conf"
)

type appResult struct {
	exitCode int
	exitMsg  string
}

type cmdOpts struct {
	config    string
	daemon    bool
	verbose   bool
	install   bool
	uninstall bool
	version   bool
	help      bool
	user      string
	pw        string
}

// Application entry point
func main() {
	result := &appResult{}
	defer exit(result)

	opts, fs, err := parseFlags(os.Args[0], os.Args[1:], os.Stderr)
	if err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			result.exitCode = 2
		}
		return
	}

	// possibly display help
	if opts.help {
		fs.PrintDefaults()
		return
	}

	// possibly display version info
	if opts.version {
		fmt.Println(appVersion)
		return
	}

	// possibly install as service
	if opts.install {
		err := serviceInstall(opts)
		if err != nil {
			result.exitCode = -5
			result.exitMsg = fmt.Sprintf("%v", err)
		} else {
			result.exitMsg = "install successful"
		}
		return
	}

	// possibly uninstall
	if opts.uninstall {
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
	if opts.daemon {
		err := serviceRun(opts.config)
		if err != nil {
			result.exitCode = -1
			result.exitMsg = fmt.Sprintf("%v", err)
		}
		return
	}

	res, err := runOnce(opts.config, opts.verbose)
	result.exitMsg = resultMessage(res, err)
	if err != nil {
		result.exitCode = -1
	}
}

func parseFlags(name string, args []string, output io.Writer) (*cmdOpts, *flag.FlagSet, error) {
	opts := &cmdOpts{}
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(output)
	fs.StringVar(&opts.config, "f", defConfigFile(), "config file")
	fs.BoolVar(&opts.daemon, "d", false, "run as a daemon")
	fs.BoolVar(&opts.verbose, "v", false, "overrides verbose setting in config (interactive only)")
	fs.BoolVar(&opts.install, "i", false, "install as service/daemon")
	fs.BoolVar(&opts.uninstall, "u", false, "uninstall service/daemon")
	fs.BoolVar(&opts.version, "version", false, "display version info")
	fs.BoolVar(&opts.help, "h", false, "display help")
	fs.StringVar(&opts.user, "user", "", "user account name to run service (with -i)")
	fs.StringVar(&opts.pw, "pw", "", "password of user account to run service (with -i)")
	err := fs.Parse(args)
	return opts, fs, err
}

func runOnce(fileConfig string, verbose bool) (Result, error) {
	// load config file
	appConfig, err := NewAppConfig(fileConfig)
	if err != nil {
		return LOCALERROR, err
	}

	// if verbose specified on command line it overrides config
	if verbose {
		src := cfg.NewSrcMapFromMap(map[string]string{"verbose": "YES"})
		appConfig.PrependSource(src)
	}

	// configure logger
	logger, closeLog, err := configureLogging(appConfig)
	if err != nil {
		return LOCALERROR, err
	}
	defer closeLog()
	return updateIP(appConfig, logger)
}

func resultMessage(res Result, err error) string {
	if err == nil || err.Error() == string(res) {
		return string(res)
	}
	return fmt.Sprintf("%s: %v", res, err)
}

func configureLogging(cfg *AppConfig) (*logrus.Logger, func(), error) {
	file := cfg.getKeyVal(keyLogFile)
	syslogger := isTrue(cfg.getKeyVal(keySyslog))
	verbose := isTrue(cfg.getKeyVal(keyVerbose))

	logger := log.New()
	logger.Level = log.InfoLevel
	logger.Out = os.Stdout
	closeLog := func() {}

	switch {
	case syslogger:
		if !hasSysLog() {
			return nil, nil, fmt.Errorf("syslog not supported for %s", runtime.GOOS)
		}
		hook, err := syslogHook("dynip")
		if err != nil {
			return nil, nil, err
		}
		if hook != nil {
			logger.AddHook(hook)
		}
		logger.Out = io.Discard
	case file != "":
		f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, nil, err
		}
		logger.Out = f
		closeLog = func() { _ = f.Close() }
	}
	if verbose {
		logger.Level = log.DebugLevel
	}
	return logger, closeLog, nil
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
