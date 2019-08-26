package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kardianos/service"
)

type srvOpt struct {
	config string
	user   string
	pw     string
}

func serviceInstall() error {
	var err error
	opt, err := parseCmdLine()

	cfg := serviceConfig()
	cfg.Arguments = serviceCmdLineArgs(opt)
	if err != nil {
		return fmt.Errorf("service install args: %v", err)
	}
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

func serviceRun() error {
	cfg := serviceConfig()
	prg := &program{}
	srv, err := service.New(prg, cfg)
	if err != nil {
		return err
	}
	return srv.Run()
}

func serviceConfig() *service.Config {
	cfg := &service.Config{
		Name:        "dynip",
		DisplayName: "dynip",
		Description: "Dynamic IP update service",
		Option:      make(map[string]interface{}),
	}
	return cfg
}

func serviceCmdLineArgs(opt *srvOpt) []string {
	arr := []string{"-d"}
	if opt.config != "" {
		arr = append(arr, fmt.Sprintf("-f %s", opt.config))
	}
	return arr
}

func parseCmdLine() (*srvOpt, error) {
	opt := srvOpt{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&opt.config, "f", defConfigFile(), "config file")
	fs.StringVar(&opt.user, "user", "", "user account name to run service")
	fs.StringVar(&opt.pw, "pw", "", "password of user account to run service")
	err := fs.Parse(os.Args[1:])
	return &opt, err
}
