package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/kardianos/service"
)

// stopTimeout bounds how long Stop waits for an in-flight update; service
// managers expect Stop to return within a few seconds.
const stopTimeout = 5 * time.Second

type program struct {
	configFile string
	exit       chan struct{}
	done       chan struct{}
	logger     *logrus.Logger
	closeLog   func()
}

// Start is called by service manager to start the service. Don't block.
func (p *program) Start(s service.Service) (err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Panic in Start: ", r)
			err = fmt.Errorf("panic in Start: %v", r)
		}
	}()

	appConfig, err := NewAppConfig(p.configFile)
	if err != nil {
		return err
	}

	p.logger, p.closeLog, err = configureLogging(appConfig)
	if err != nil {
		return err
	}

	p.exit = make(chan struct{})
	p.done = make(chan struct{})
	do := &daemonOpt{appConfig: appConfig, logger: p.logger, exit: p.exit, done: p.done}
	go runDaemon(do)

	return nil
}

// Stop is called by service manager to stop the service. Don't block for more
// than a few seconds.
func (p *program) Stop(s service.Service) error {
	if p.exit != nil {
		close(p.exit)
		select {
		case <-p.done:
		case <-time.After(stopTimeout):
		}
	}
	if p.closeLog != nil {
		p.closeLog()
	}
	return nil
}
