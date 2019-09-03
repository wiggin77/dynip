package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/kardianos/service"
)

type program struct {
	exit   chan string
	logger *logrus.Logger
}

// Start is called by service manager to start the service. Don't block.
func (p *program) Start(s service.Service) error {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Panic in Start: ", r)
		}
	}()

	p.exit = make(chan string, 2)

	// get config file name
	var file string
	var daemon bool
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&file, "f", defConfigFile(), "config file")
	fs.BoolVar(&daemon, "d", false, "run as a daemon")
	err := fs.Parse(os.Args[1:])
	if err != nil {
		return err
	}

	// load config file
	appConfig, err := NewAppConfig(file)
	if err != nil {
		return err
	}

	// configure logger
	p.logger, err = configureLogging(appConfig)
	if err != nil {
		return err
	}

	do := &daemonOpt{appConfig: appConfig, logger: p.logger, exit: p.exit}
	go runDaemon(do)

	return nil
}

// Stop is called by service manager to stop the service. Don't block for more
// than a few seconds.
func (p *program) Stop(s service.Service) error {
	// close the log file (if any)
	if p.logger != nil {
		c, ok := p.logger.Out.(io.Closer)
		if ok {
			_ = c.Close()
		}
	}
	// Stop should not block. Return within a few seconds.
	p.exit <- "service controller issued Stop command"
	close(p.exit)
	return nil
}
