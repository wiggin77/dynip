package main

import (
	"flag"
	"fmt"
	"os"
	"os/user"
	"path"
	"runtime/debug"
)

// Default config file directory and name.
const (
	defaultConfigDir  = ".dynip"
	defaultConfigFile = "dynip.conf"
)

// Global `cfg.Config`
var appConfig *AppConfig

// Config returns the global `cfg.Config`
func Config() *AppConfig {
	if appConfig == nil {
		panic(fmt.Errorf("appConfig not initialized"))
	}
	return appConfig
}

// Application entry point
func main() {
	var exitCode int
	var exitMsg string
	defer exit(&exitCode, &exitMsg)

	var fileConfig string
	var daemon bool
	defFileConfig := defConfigFile()

	// Process command line flags
	flag.StringVar(&fileConfig, "f", defFileConfig, "config file")
	flag.BoolVar(&daemon, "d", false, "run as daemon")
	flag.Parse()

	// Load config file
	var err error
	appConfig, err = NewAppConfig(fileConfig)
	if err != nil {
		exitCode = -1
		exitMsg = fmt.Sprintf("%v", err)
		return
	}

	// Run...
}

// Get the filespec for the default config file in user's home directory.
func defConfigFile() string {
	home, err := homePath()
	if err == nil {
		home = path.Join(home, defaultConfigDir, defaultConfigFile)
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

// Exit app with return code and optional error message.
func exit(code *int, msg *string) {

	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "Panic: %s\n%s", r, debug.Stack())
		if *code == 0 {
			*code = -1
		}
	}

	if len(*msg) > 0 {
		fmt.Fprintf(os.Stderr, "%s\n", *msg)
	} else {
		fmt.Printf("exiting with code %d", *code)
	}

	os.Exit(*code)
}
