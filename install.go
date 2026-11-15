package main

import (
	"fmt"
	"path/filepath"

	"github.com/kardianos/service"
)

func serviceInstall(opt *cmdOpts) error {
	args, err := serviceCmdLineArgs(opt.config)
	if err != nil {
		return fmt.Errorf("service install: %v", err)
	}
	cfg := serviceConfig()
	cfg.Arguments = args
	cfg.UserName = opt.user
	if opt.pw != "" {
		cfg.Option["Password"] = opt.pw
	}
	prg := &program{}
	srv, err := service.New(prg, cfg)
	if err != nil {
		return fmt.Errorf("service install creater: %v", err)
	}
	err = srv.Install()
	if err != nil {
		return fmt.Errorf("service install: %v", err)
	}
	return nil
}

func serviceUninstall() error {
	cfg := serviceConfig()
	prg := &program{}
	srv, err := service.New(prg, cfg)
	if err != nil {
		return err
	}
	return srv.Uninstall()
}

func serviceRun(configFile string) error {
	cfg := serviceConfig()
	prg := &program{configFile: configFile}
	srv, err := service.New(prg, cfg)
	if err != nil {
		return err
	}
	return srv.Run() // blocks until Stop()
}

func serviceConfig() *service.Config {
	cfg := &service.Config{
		Name:        "dynip",
		DisplayName: "dynip",
		Description: "Dynamic IP update service",
		Option:      make(map[string]any),
	}
	return cfg
}

// serviceCmdLineArgs makes configFile absolute because the service manager
// does not start the service in the installer's working directory.
func serviceCmdLineArgs(configFile string) ([]string, error) {
	arr := []string{"-d"}
	if configFile != "" {
		abs, err := filepath.Abs(configFile)
		if err != nil {
			return nil, err
		}
		arr = append(arr, "-f", abs)
	}
	return arr, nil
}
