package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestBackoff(t *testing.T) {
	bo := &backoff{maxSkips: 2}
	step := func(wantReady bool) {
		t.Helper()
		if got := bo.ready(); got != wantReady {
			t.Fatalf("ready() = %v, want %v (state %+v)", got, wantReady, *bo)
		}
	}

	step(true)
	bo.record(true)
	step(false)
	step(true)
	bo.record(true)
	step(false)
	step(false)
	step(true)
	bo.record(true) // at maxSkips; skip stays 2
	step(false)
	step(false)
	step(true)
	bo.record(false)
	step(true)
	step(true)
}

func countingServer(t *testing.T, body string) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	hits := make(chan struct{}, 10)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, body)
		hits <- struct{}{}
	}))
	t.Cleanup(ts.Close)
	return ts, hits
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func TestRunDaemonUpdatesImmediatelyAndExits(t *testing.T) {
	ts, hits := countingServer(t, respSUCCESS)
	cfg, err := makeTestConfig(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	logger := logrus.New()
	logger.Out = io.Discard

	exit := make(chan struct{})
	done := make(chan struct{})
	go runDaemon(&daemonOpt{appConfig: cfg, logger: logger, exit: exit, done: done})

	waitFor(t, hits, "immediate update")
	close(exit)
	waitFor(t, done, "daemon exit")
}

func TestProgramStartStop(t *testing.T) {
	ts, hits := countingServer(t, respSUCCESS)
	logFile := filepath.Join(t.TempDir(), "dynip.log")
	props := requiredProps()
	props["proto"] = "http"
	props["url"] = strings.TrimPrefix(ts.URL, "http://")
	props["log"] = logFile

	p := &program{configFile: writeConfigFile(t, props)}
	if err := p.Start(nil); err != nil {
		t.Fatal(err)
	}
	waitFor(t, hits, "immediate update")
	if err := p.Stop(nil); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ip update successful", "Dynip daemon exiting"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("log missing %q; got %q", want, data)
		}
	}
}

func TestProgramStartBadConfig(t *testing.T) {
	p := &program{configFile: filepath.Join(t.TempDir(), "nope.conf")}
	if err := p.Start(nil); err == nil {
		t.Fatal("expected error for missing config file")
	}
	if err := p.Stop(nil); err != nil {
		t.Fatalf("Stop after failed Start: %v", err)
	}
}
