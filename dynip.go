package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func updateIP(appConfig *AppConfig, logger *logrus.Logger) (Result, error) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			logger.Panicf("Panic: %s\n%s", r, debug.Stack())
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

	url := makeURL(appConfig)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return LOCALERROR, err
	}

	log.Debug("request: ", url)

	resp, err := client.Do(req)
	if err != nil {
		return LOCALERROR, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
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

func makeURL(appConfig *AppConfig) string {
	var sb strings.Builder

	sb.WriteString(appConfig.getKeyVal(keyProto))
	sb.WriteString("://")
	sb.WriteString(appConfig.getKeyVal(keyUsername))
	sb.WriteString(":")
	sb.WriteString(appConfig.getKeyVal(keyToken))
	sb.WriteString("@")
	sb.WriteString(appConfig.getKeyVal(keyURL))
	sb.WriteString("?")

	keys := appConfig.getKeys()
	for _, k := range keys {
		val := appConfig.getKeyVal(k)
		if shouldInc(k.inc, val) {
			sb.WriteString(k.name)
			sb.WriteString("=")
			sb.WriteString(val)
			sb.WriteString("&")
		}
	}
	return sb.String()
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
	// SERVERERROR means a generic error occured on the server
	SERVERERROR Result = "SERVER_ERROR"
	// UNKNOWN means the response contained none of the known result codes
	UNKNOWN Result = "UNKNOWN_RESPONSE"
	// LOCALERROR means there was a local error
	LOCALERROR Result = "LOCAL_ERROR"
)
