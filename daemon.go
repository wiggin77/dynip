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
	hostname := appConfig.getKeyVal(keyHostname)

	log := logger.WithFields(logrus.Fields{"interval": dur, "hostname": hostname})
	log.Info("Dynip daemon starting")

	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	go signalMon(exit)

	var err error
	var result Result
	var skip, skipCount int
	var maxSkips = int((time.Hour * 24) / dur)
	for {
		select {
		case msg := <-exit:
			log.Info("Dynip daemon exiting: ", msg)
			return nil
		case <-ticker.C:
			if skipCount >= skip {
				skipCount = 0
				log.Info("Dynip updating IP")
				result, err = updateIP(appConfig, logger)
				if err == nil {
					skip = 0
					log.WithFields(logrus.Fields{"result": result}).Info("ip update successful")
				} else if skip < maxSkips {
					skip++
					log.WithFields(logrus.Fields{"result": result, "err": err}).Error("ip update failed")
				}
			} else {
				skipCount++
				log.WithFields(logrus.Fields{
					"skips_remaining": skip - skipCount}).Info("Skipping due to previous errors")
			}
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
