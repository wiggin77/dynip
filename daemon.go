package main

import (
	"time"

	"github.com/sirupsen/logrus"
)

type daemonOpt struct {
	appConfig *AppConfig
	logger    *logrus.Logger
	exit      <-chan struct{}
	done      chan<- struct{}
}

// backoff skips one more tick after each consecutive failure, up to maxSkips.
type backoff struct {
	skip      int
	skipCount int
	maxSkips  int
}

func (b *backoff) ready() bool {
	if b.skipCount >= b.skip {
		b.skipCount = 0
		return true
	}
	b.skipCount++
	return false
}

func (b *backoff) record(failed bool) {
	if !failed {
		b.skip = 0
	} else if b.skip < b.maxSkips {
		b.skip++
	}
}

func (b *backoff) remaining() int {
	return b.skip - b.skipCount
}

// runDaemon updates the IP immediately, then on each interval tick until
// daemonOpt.exit is closed. It closes daemonOpt.done on return.
func runDaemon(do *daemonOpt) {
	defer close(do.done)

	dur, _ := do.appConfig.Duration(keyInterval.name, time.Minute*11)
	if dur <= time.Second {
		do.logger.Errorf("invalid interval (%v); defaulting to 11 minutes", dur)
		dur = time.Minute * 11
	}
	hostname := do.appConfig.getKeyVal(keyHostname)

	log := do.logger.WithFields(logrus.Fields{"interval": dur, "hostname": hostname})
	log.Info("Dynip daemon starting")

	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	bo := &backoff{maxSkips: int((time.Hour * 24) / dur)}
	update := func() {
		if !bo.ready() {
			log.WithFields(logrus.Fields{"skips_remaining": bo.remaining()}).Info("Skipping due to previous errors")
			return
		}
		log.Info("Dynip updating IP")
		result, err := updateIP(do.appConfig, do.logger)
		bo.record(err != nil)
		if err != nil {
			log.WithFields(logrus.Fields{"result": result, "err": err}).Error("ip update failed")
			return
		}
		log.WithFields(logrus.Fields{"result": result}).Info("ip update successful")
	}

	update()
	for {
		select {
		case <-do.exit:
			log.Info("Dynip daemon exiting")
			return
		case <-ticker.C:
			update()
		}
	}
}
