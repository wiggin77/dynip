package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseFlags(t *testing.T) {
	t.Run("windows install with user", func(t *testing.T) {
		opts, _, err := parseFlags("dynip", []string{"-i", "-f", `c:\dynip\dynip.conf`, "-user", "bob", "-pw", "secret"}, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		want := cmdOpts{install: true, config: `c:\dynip\dynip.conf`, user: "bob", pw: "secret"}
		if *opts != want {
			t.Errorf("parseFlags() = %+v, want %+v", *opts, want)
		}
	})

	t.Run("daemon with verbose", func(t *testing.T) {
		opts, _, err := parseFlags("dynip", []string{"-d", "-v", "-f", "x.conf"}, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		want := cmdOpts{daemon: true, verbose: true, config: "x.conf"}
		if *opts != want {
			t.Errorf("parseFlags() = %+v, want %+v", *opts, want)
		}
	})

	t.Run("defaults", func(t *testing.T) {
		opts, _, err := parseFlags("dynip", nil, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		want := cmdOpts{config: defConfigFile()}
		if *opts != want {
			t.Errorf("parseFlags() = %+v, want %+v", *opts, want)
		}
	})

	t.Run("unknown flag", func(t *testing.T) {
		_, _, err := parseFlags("dynip", []string{"-bogus"}, io.Discard)
		if err == nil || errors.Is(err, flag.ErrHelp) {
			t.Fatalf("err = %v, want unknown flag error", err)
		}
	})

	t.Run("help", func(t *testing.T) {
		_, _, err := parseFlags("dynip", []string{"-help"}, io.Discard)
		if !errors.Is(err, flag.ErrHelp) {
			t.Fatalf("err = %v, want flag.ErrHelp", err)
		}
	})
}

func TestResultMessage(t *testing.T) {
	tests := []struct {
		name string
		res  Result
		err  error
		want string
	}{
		{name: "success", res: SUCCESS, want: "SUCCESS"},
		{name: "no change", res: NOCHANGE, want: "NO_CHANGE"},
		{name: "remote error", res: NOAUTH, err: fmt.Errorf("%v", NOAUTH), want: "NO_AUTH"},
		{name: "local error", res: LOCALERROR, err: errors.New("connection refused"), want: "LOCAL_ERROR: connection refused"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resultMessage(tt.res, tt.err); got != tt.want {
				t.Errorf("resultMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfigureLogging(t *testing.T) {
	t.Run("defaults to stdout at info", func(t *testing.T) {
		cfg, err := NewAppConfigFromMap(requiredProps())
		if err != nil {
			t.Fatal(err)
		}
		logger, closeLog, err := configureLogging(cfg)
		if err != nil {
			t.Fatal(err)
		}
		closeLog()
		if logger.Out != os.Stdout {
			t.Errorf("Out = %v, want os.Stdout", logger.Out)
		}
		if logger.Level != logrus.InfoLevel {
			t.Errorf("Level = %v, want info", logger.Level)
		}
		if _, err := fmt.Fprint(os.Stdout, ""); err != nil {
			t.Errorf("closeLog closed stdout: %v", err)
		}
	})

	t.Run("verbose sets debug", func(t *testing.T) {
		props := requiredProps()
		props["verbose"] = "YES"
		cfg, err := NewAppConfigFromMap(props)
		if err != nil {
			t.Fatal(err)
		}
		logger, closeLog, err := configureLogging(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer closeLog()
		if logger.Level != logrus.DebugLevel {
			t.Errorf("Level = %v, want debug", logger.Level)
		}
	})

	t.Run("log file appends", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "dynip.log")
		props := requiredProps()
		props["log"] = file
		cfg, err := NewAppConfigFromMap(props)
		if err != nil {
			t.Fatal(err)
		}
		for _, msg := range []string{"first run", "second run"} {
			logger, closeLog, err := configureLogging(cfg)
			if err != nil {
				t.Fatal(err)
			}
			logger.Info(msg)
			closeLog()
		}

		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "first run") || !strings.Contains(string(data), "second run") {
			t.Errorf("log file contents = %q, want both runs", data)
		}
	})

	t.Run("log file in missing directory", func(t *testing.T) {
		props := requiredProps()
		props["log"] = filepath.Join(t.TempDir(), "missing", "dynip.log")
		cfg, err := NewAppConfigFromMap(props)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := configureLogging(cfg); err == nil {
			t.Fatal("expected error for unwritable log path")
		}
	})

	t.Run("syslog", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "dynip.log")
		props := requiredProps()
		props["syslog"] = "YES"
		props["log"] = file
		cfg, err := NewAppConfigFromMap(props)
		if err != nil {
			t.Fatal(err)
		}
		logger, closeLog, err := configureLogging(cfg)
		if !hasSysLog() {
			if err == nil {
				t.Fatal("expected error on platform without syslog")
			}
			return
		}
		if err != nil {
			t.Skipf("no local syslog daemon available: %v", err)
		}
		defer closeLog()
		if logger.Out != io.Discard {
			t.Errorf("Out = %v, want io.Discard", logger.Out)
		}
		if _, err := os.Stat(file); !os.IsNotExist(err) {
			t.Errorf("log file opened even though syslog takes precedence (stat err = %v)", err)
		}
	})
}

func startTestServer(t *testing.T, body string) string {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, body)
	}))
	t.Cleanup(ts.Close)
	uri, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	return uri.Host
}

func TestRunOnce(t *testing.T) {
	runOnceConfig := func(t *testing.T, body string) (cfgFile, logFile string) {
		t.Helper()
		logFile = filepath.Join(t.TempDir(), "dynip.log")
		props := requiredProps()
		props["proto"] = "http"
		props["url"] = startTestServer(t, body)
		props["log"] = logFile
		return writeConfigFile(t, props), logFile
	}

	t.Run("success", func(t *testing.T) {
		cfgFile, logFile := runOnceConfig(t, respSUCCESS)
		res, err := runOnce(cfgFile, false)
		if err != nil {
			t.Fatal(err)
		}
		if res != SUCCESS {
			t.Errorf("runOnce() = %v, want %v", res, SUCCESS)
		}
		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "request:") {
			t.Errorf("debug output logged without verbose: %q", data)
		}
	})

	t.Run("verbose flag overrides config", func(t *testing.T) {
		cfgFile, logFile := runOnceConfig(t, respNOCHANGE)
		res, err := runOnce(cfgFile, true)
		if err != nil {
			t.Fatal(err)
		}
		if res != NOCHANGE {
			t.Errorf("runOnce() = %v, want %v", res, NOCHANGE)
		}
		data, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "request:") || !strings.Contains(string(data), "response:") {
			t.Errorf("expected debug request/response in log, got %q", data)
		}
		if strings.Contains(string(data), "testtoken") {
			t.Errorf("token written to log: %q", data)
		}
	})

	t.Run("update failure", func(t *testing.T) {
		cfgFile, _ := runOnceConfig(t, respBADTOKEN)
		res, err := runOnce(cfgFile, false)
		if err == nil || res != NOAUTH {
			t.Fatalf("runOnce() = (%v, %v), want %s error", res, err, NOAUTH)
		}
	})

	t.Run("missing config file", func(t *testing.T) {
		res, err := runOnce(filepath.Join(t.TempDir(), "nope.conf"), false)
		if err == nil || res != LOCALERROR {
			t.Fatalf("runOnce() = (%v, %v), want %s error", res, err, LOCALERROR)
		}
	})

	t.Run("bad log path", func(t *testing.T) {
		props := requiredProps()
		props["log"] = filepath.Join(t.TempDir(), "missing", "dynip.log")
		res, err := runOnce(writeConfigFile(t, props), false)
		if err == nil || res != LOCALERROR {
			t.Fatalf("runOnce() = (%v, %v), want %s error", res, err, LOCALERROR)
		}
	})
}
