package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

func updateIP(appConfig *AppConfig, logger *logrus.Logger) error {

	timeout := time.Second * 90
	client := http.Client{
		Timeout: timeout,
	}

	url := makeURL(appConfig)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	logger.Debug("request: ", url)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	bodyTxt := string(body)
	logger.Debug("response: ", bodyTxt)

	success, code := parseResponse(bodyTxt)
	if !success {
		return fmt.Errorf(code)
	}

	logger.Info("ip update: ", code)
	return nil
}

func makeURL(appConfig *AppConfig) string {
	var sb strings.Builder

	sb.WriteString("https://")
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

func isFalse(s string) bool {
	s = strings.ToUpper(s)
	return s == "NO" || s == "OFF" || s == "FALSE"
}

func parseResponse(s string) (success bool, code string) {
	if strings.Contains(s, "NOERROR") {
		return true, "SUCCESS"
	}
	if strings.Contains(s, "OK") {
		return true, "NO_CHANGE"
	}
	if strings.Contains(s, "NOACCESS") {
		return false, "NO_ACCESS"
	}
	if strings.Contains(s, "NOSERVICE") || strings.Contains(s, "NO_SERVICE") {
		return false, "NO_SERVICE"
	}
	if strings.Contains(s, "ILLEGAL INPUT") {
		return false, "ILLEGAL_INPUT"
	}
	if strings.Contains(s, "TOOSOON") || strings.Contains(s, "TOO_FREQ") {
		return false, "TOO_SOON"
	}
	return false, "UKNOWN_RESPONSE"
}
