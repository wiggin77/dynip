package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

func runDaemon(appConfig *AppConfig, logger *logrus.Logger, exit chan string) error {
	dur, _ := appConfig.Duration(keyInterval.name, time.Minute*11)
	if dur <= time.Second {
		logger.Errorf("invalid interval (%v); defaulting to 11 minutes", dur)
		dur = time.Minute * 11
	}
	hostname := appConfig.Hostname()

	logger.WithFields(logrus.Fields{"interval": dur, "hostname": hostname}).Info("Dynip daemon starting")

	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	go signalMon(exit)

	for {
		select {
		case msg := <-exit:
			logger.Info("Dynip daemon exiting: ", msg)
			return nil
		case <-ticker.C:
			logger.WithField("hostname", hostname).Info("Dynip updating IP")
			updateIP(appConfig, logger)
		}
	}
}

func signalMon(exit chan<- string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(c)

	// block until signal
	s := <-c
	exit <- fmt.Sprintf("%v", s)
}
