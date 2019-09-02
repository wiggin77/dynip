// +build windows nacl plan9

package main

import (
	"github.com/sirupsen/logrus"
)

// syslogHook returns nil on Windows as syslog is not supported.
func syslogHook(tag string) (logrus.Hook, error) {
	return nil, nil
}
