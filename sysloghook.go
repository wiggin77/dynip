// +build !windows,!nacl,!plan9

package main

import (
	"log/syslog"

	"github.com/sirupsen/logrus"
	logrussyslog "github.com/sirupsen/logrus/hooks/syslog"
)

// syslogHook returns a logrus syslog hook on supported platforms.
func syslogHook(tag string) (logrus.Hook, error) {
	return logrussyslog.NewSyslogHook("", "", syslog.LOG_INFO, tag)
}
