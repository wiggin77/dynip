package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kardianos/service"
)

type srvOpt struct {
	install bool
	config  string
	user    string
	pw      string
}

func serviceInstall() error {
	var err error
	opt, err := parseCmdLine()
	if err != nil {
		return err
	}

	cfg := serviceConfig()
	cfg.Arguments = serviceCmdLineArgs(opt)
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
	return srv.Run() // blocks until Stop()
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
		arr = append(arr, "-f")
		arr = append(arr, opt.config)
	}
	return arr
}

func parseCmdLine() (*srvOpt, error) {
	opt := srvOpt{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&opt.install, "i", false, "install as service/daemon")
	fs.StringVar(&opt.config, "f", defConfigFile(), "config file")
	fs.StringVar(&opt.user, "user", "", "user account name to run service")
	fs.StringVar(&opt.pw, "pw", "", "password of user account to run service")
	err := fs.Parse(os.Args[1:])
	return &opt, err
}
