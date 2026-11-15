package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// updateIP makes one HTTP(S) request to Dynamic IP server then returns the result.
func updateIP(appConfig *AppConfig, logger *logrus.Logger) (result Result, err error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Panic: %v\n%s", r, debug.Stack())
			result = LOCALERROR
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	log := logger.WithField("hostname", appConfig.getKeyVal(keyHostname))
	timeout := time.Second * 90
	client := http.Client{
		Timeout: timeout,
	}

	u, err := makeURL(appConfig)
	if err != nil {
		return LOCALERROR, err
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return LOCALERROR, err
	}

	log.Debug("request: ", u.Redacted())

	resp, err := client.Do(req)
	if err != nil {
		return LOCALERROR, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return LOCALERROR, err
	}

	bodyTxt := string(body)
	log.Debug("response: ", bodyTxt)

	success, result := parseResponse(bodyTxt)
	if !success {
		return result, fmt.Errorf("%v", result)
	}
	return result, nil
}

// makeURL parses only proto and url before adding credentials, so a parse
// error cannot leak the token.
func makeURL(appConfig *AppConfig) (*url.URL, error) {
	u, err := url.Parse(appConfig.getKeyVal(keyProto) + "://" + appConfig.getKeyVal(keyURL))
	if err != nil {
		return nil, err
	}
	u.User = url.UserPassword(appConfig.getKeyVal(keyUsername), appConfig.getKeyVal(keyToken))

	q := url.Values{}
	for _, k := range appConfig.getKeys() {
		val := appConfig.getKeyVal(k)
		if shouldInc(k.inc, val) {
			q.Set(k.name, val)
		}
	}
	u.RawQuery = q.Encode()
	return u, nil
}

func shouldInc(rule incRule, val string) bool {
	switch rule {
	case NEVER:
		return false
	case ALWAYS:
		return true
	case NOTEMPTY:
		return len(val) > 0
	case NOTFALSE:
		return !isFalse(val)
	default:
		return false
	}
}

func parseResponse(s string) (success bool, result Result) {
	if strings.Contains(s, ">OK<") || strings.Contains(s, ">NOERROR<") {
		if strings.Contains(s, " updated to ") {
			return true, SUCCESS
		}
		return true, NOCHANGE
	}
	if strings.Contains(s, ">NO_AUTH<") || strings.Contains(s, ">NOACCESS<") {
		return false, NOAUTH
	}
	if strings.Contains(s, ">NOSERVICE<") || strings.Contains(s, ">NO_SERVICE<") {
		return false, NOSERVICE
	}
	if strings.Contains(s, ">ILLEGAL<") || strings.Contains(s, ">ILLEGAL_INPUT<") {
		return false, ILLEGALINPUT
	}
	if strings.Contains(s, ">TOOSOON<") || strings.Contains(s, ">TOO_FREQ<") {
		return false, TOOSOON
	}
	if strings.Contains(s, ">NO_PARTNER<") || strings.Contains(s, ">NOPARTNER<") {
		return false, NOPARTNER
	}
	if strings.Contains(s, ">ERROR<") {
		return false, SERVERERROR
	}
	return false, UNKNOWN
}

// Result represents the result code returned from an IP update request.
type Result string

const (
	// SUCCESS means the IP address updated successfully
	SUCCESS Result = "SUCCESS"
	// NOCHANGE means IP address is already the requested value
	NOCHANGE Result = "NO_CHANGE"
	// NOAUTH means password/token incorrect
	NOAUTH Result = "NO_AUTH"
	// NOSERVICE means Dynamic DNS is not turned on for this domain
	NOSERVICE Result = "NO_SERVICE"
	// ILLEGALINPUT means a request param was invalid
	ILLEGALINPUT Result = "ILLEGAL_INPUT"
	// TOOSOON means the request was issued before minimum interval elapsed
	TOOSOON Result = "TOO_SOON"
	// NOPARTNER means the request did not include required partner information
	// or there was an error detecting the partner.
	NOPARTNER Result = "NO_PARTNER"
	// SERVERERROR means a generic error occurred on the server
	SERVERERROR Result = "SERVER_ERROR"
	// UNKNOWN means the response contained none of the known result codes
	UNKNOWN Result = "UNKNOWN_RESPONSE"
	// LOCALERROR means there was a local error
	LOCALERROR Result = "LOCAL_ERROR"
)
