package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
)

var response string

func Test_updateIP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, response)
	}))
	defer ts.Close()

	type args struct {
		appConfig *AppConfig
		logger    *logrus.Logger
	}

	cfg, err := makeTestConfig(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	tlog := logrus.New()
	tlog.Level = logrus.InfoLevel
	tlog.Out = io.Discard
	targs := args{cfg, tlog}

	tests := []struct {
		name    string
		args    args
		want    Result
		wantErr bool
		resp    string
	}{
		{name: "success", args: targs, want: SUCCESS, wantErr: false, resp: respSUCCESS},
		{name: "no change", args: targs, want: NOCHANGE, wantErr: false, resp: respNOCHANGE},
		{name: "too soon", args: targs, want: TOOSOON, wantErr: true, resp: respTOOSOON},
		{name: "bad token", args: targs, want: NOAUTH, wantErr: true, resp: respBADTOKEN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response = tt.resp
			got, err := updateIP(tt.args.appConfig, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateIP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("updateIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateIPLocalErrors(t *testing.T) {
	tlog := logrus.New()
	tlog.Out = io.Discard

	closed := httptest.NewServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()

	truncated := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = fmt.Fprint(w, "short")
	}))
	defer truncated.Close()

	tests := []struct {
		name string
		cfg  func(t *testing.T) *AppConfig
	}{
		{name: "invalid url", cfg: func(t *testing.T) *AppConfig {
			cfg, err := NewAppConfigFromMap(map[string]string{"url": "%zz", "proto": "http",
				"hostname": "test.example.com", "username": "testuser", "token": "testtoken"})
			if err != nil {
				t.Fatal(err)
			}
			return cfg
		}},
		{name: "connection refused", cfg: func(t *testing.T) *AppConfig {
			cfg, err := makeTestConfig(closedURL)
			if err != nil {
				t.Fatal(err)
			}
			return cfg
		}},
		{name: "truncated body", cfg: func(t *testing.T) *AppConfig {
			cfg, err := makeTestConfig(truncated.URL)
			if err != nil {
				t.Fatal(err)
			}
			return cfg
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := updateIP(tt.cfg(t), tlog)
			if err == nil {
				t.Fatal("expected error")
			}
			if strings.Contains(err.Error(), "testtoken") {
				t.Errorf("error leaks token: %v", err)
			}
			if got != LOCALERROR {
				t.Errorf("updateIP() = %v, want %v", got, LOCALERROR)
			}
		})
	}
}

func Test_updateIPRecoversPanic(t *testing.T) {
	var buf bytes.Buffer
	tlog := logrus.New()
	tlog.Out = &buf

	got, err := updateIP(nil, tlog)
	if err == nil {
		t.Fatal("expected error from recovered panic")
	}
	if got != LOCALERROR {
		t.Errorf("updateIP() = %v, want %v", got, LOCALERROR)
	}
	if !strings.Contains(buf.String(), "Panic:") {
		t.Errorf("panic not logged, got %q", buf.String())
	}
}

func Test_makeURLEscapes(t *testing.T) {
	const token = "p@ss:w/rd#1?&"
	cfg, err := NewAppConfigFromMap(map[string]string{
		"username": "test user",
		"token":    token,
		"hostname": "test.example.com",
		"mx":       "mx.example.com&wildcard=ON",
	})
	if err != nil {
		t.Fatal(err)
	}
	u, err := makeURL(cfg)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(u.String())
	if err != nil {
		t.Fatalf("makeURL produced unparseable URL: %v", err)
	}
	if got := parsed.User.Username(); got != "test user" {
		t.Errorf("username = %q", got)
	}
	if got, _ := parsed.User.Password(); got != token {
		t.Errorf("token = %q, want %q", got, token)
	}
	if parsed.Scheme != "https" || parsed.Host != "api.cp.easydns.com" || parsed.Path != "/dyn/generic.php" {
		t.Errorf("unexpected endpoint %q", parsed.Redacted())
	}

	q := parsed.Query()
	want := url.Values{
		"hostname": {"test.example.com"},
		"myip":     {"1.1.1.1"},
		"mx":       {"mx.example.com&wildcard=ON"},
	}
	if q.Encode() != want.Encode() {
		t.Errorf("query = %v, want %v", q, want)
	}
	if strings.Contains(u.Redacted(), "p@ss") {
		t.Errorf("Redacted() leaks token: %s", u.Redacted())
	}
}

func Test_updateIPSendsCredentials(t *testing.T) {
	const token = "p@ss:w/rd#1"
	var gotUser, gotPass, gotHost string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		gotHost = r.URL.Query().Get("hostname")
		_, _ = fmt.Fprintln(w, respNOCHANGE)
	}))
	defer ts.Close()

	cfg, err := NewAppConfigFromMap(map[string]string{"url": strings.TrimPrefix(ts.URL, "http://"), "proto": "http",
		"hostname": "test.example.com", "username": "testuser", "token": token})
	if err != nil {
		t.Fatal(err)
	}
	tlog := logrus.New()
	tlog.Out = io.Discard
	if _, err := updateIP(cfg, tlog); err != nil {
		t.Fatal(err)
	}
	if gotUser != "testuser" || gotPass != token || gotHost != "test.example.com" {
		t.Errorf("server saw user=%q pass=%q hostname=%q", gotUser, gotPass, gotHost)
	}
}

func Test_parseResponse(t *testing.T) {
	tests := []struct {
		name        string
		resp        string
		wantSuccess bool
		want        Result
	}{
		{name: "ok updated", resp: ">OK< host updated to 1.2.3.4", wantSuccess: true, want: SUCCESS},
		{name: "ok no change", resp: ">OK< no update required", wantSuccess: true, want: NOCHANGE},
		{name: "noerror updated", resp: ">NOERROR< host updated to 1.2.3.4", wantSuccess: true, want: SUCCESS},
		{name: "noerror no change", resp: ">NOERROR<", wantSuccess: true, want: NOCHANGE},
		{name: "no_auth", resp: ">NO_AUTH<", want: NOAUTH},
		{name: "noaccess", resp: ">NOACCESS<", want: NOAUTH},
		{name: "noservice", resp: ">NOSERVICE<", want: NOSERVICE},
		{name: "no_service", resp: ">NO_SERVICE<", want: NOSERVICE},
		{name: "illegal", resp: ">ILLEGAL<", want: ILLEGALINPUT},
		{name: "illegal_input", resp: ">ILLEGAL_INPUT<", want: ILLEGALINPUT},
		{name: "toosoon", resp: ">TOOSOON<", want: TOOSOON},
		{name: "too_freq", resp: ">TOO_FREQ<", want: TOOSOON},
		{name: "no_partner", resp: ">NO_PARTNER<", want: NOPARTNER},
		{name: "nopartner", resp: ">NOPARTNER<", want: NOPARTNER},
		{name: "error", resp: ">ERROR<", want: SERVERERROR},
		{name: "unknown", resp: "<html>something else</html>", want: UNKNOWN},
		{name: "empty", resp: "", want: UNKNOWN},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, got := parseResponse(tt.resp)
			if success != tt.wantSuccess || got != tt.want {
				t.Errorf("parseResponse(%q) = (%v, %v), want (%v, %v)", tt.resp, success, got, tt.wantSuccess, tt.want)
			}
		})
	}
}

func makeTestConfig(surl string) (*AppConfig, error) {
	uri, err := url.Parse(surl)
	if err != nil {
		return nil, err
	}
	purl := fmt.Sprintf("%s:%s", uri.Hostname(), uri.Port())

	m := map[string]string{"url": purl,
		"proto":    "http",
		"hostname": "test.example.com",
		"username": "testuser",
		"token":    "testtoken"}
	return NewAppConfigFromMap(m)
}

const (
	respSUCCESS = `<HTML><BODY><FONT FACE="sans-serif" SIZE="-1">OK<br />
	<hr noshade size="1">
	test.example.com updated to 24.114.104.44<br />
	</FONT></BODY></HTML>`

	respNOCHANGE = `<HTML><BODY><FONT FACE="sans-serif" SIZE="-1">OK<br />
	<hr noshade size="1">
	no update required for aspen.darklake.ca to 24.114.85.179<br />
	</FONT></BODY></HTML>`

	respTOOSOON = `<HTML><BODY><FONT FACE="sans-serif" SIZE="-1">TOO_FREQ<br />
	<hr noshade size="1">
	Increase your time between updates for test.example.com to 600 seconds or more.<br />
	</FONT></BODY></HTML>`

	respBADTOKEN = `<HTML><BODY><FONT FACE="sans-serif" SIZE="-1">NO_AUTH<br />
	<hr noshade size="1">
	Failed login attempt logged: user dlauder77, host test.example.com, from 24.114.82.202<br />
	</FONT></BODY></HTML>`
)
