package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
)

var response string

func Test_updateIP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, response)
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
	tlog.Out = ioutil.Discard
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
