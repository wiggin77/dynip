package main

import (
	"fmt"
	"strings"

	"github.com/wiggin77/cfg"
)

type configKey struct {
	name string
	def  string
	req  bool
	inc  incRule
}

type incRule int

const (
	// ALWAYS means the key is always added to API query string
	ALWAYS incRule = iota
	// NEVER means the key is never added to API query string
	NEVER
	// NOTEMPTY means the key is added to API query string only when a non-empty string
	NOTEMPTY
	// NOTFALSE means the key is added to API query string only when not "NO" or "OFF"
	NOTFALSE
)

// Configuration keys
var (
	keyProtocolVersion = configKey{name: "protocol_ver", def: "1.3", req: false, inc: NEVER}
	keyURL             = configKey{name: "url", def: "api.cp.easydns.com/dyn/generic.php", req: false, inc: NEVER}
	keyUsername        = configKey{name: "username", def: "", req: true, inc: NEVER}
	keyToken           = configKey{name: "token", def: "", req: true, inc: NEVER}
	keyHostname        = configKey{name: "hostname", def: "", req: true, inc: ALWAYS}
	keyTld             = configKey{name: "tld", def: "", req: false, inc: NOTEMPTY}
	keyMyIP            = configKey{name: "myip", def: "1.1.1.1", req: false, inc: ALWAYS}
	keyMx              = configKey{name: "mx", def: "", req: false, inc: NOTEMPTY}
	keyBackMx          = configKey{name: "backmx", def: "NO", req: false, inc: NOTFALSE}
	keyWildcard        = configKey{name: "wildcard", def: "OFF", req: false, inc: NOTFALSE}
	keyInterval        = configKey{name: "interval", def: "11 minutes", req: false, inc: NEVER}
	keyLogFile         = configKey{name: "log", def: "", req: false, inc: NEVER}
	keySyslog          = configKey{name: "syslog", def: "NO", req: false, inc: NEVER}
	keyDebug           = configKey{name: "debug", def: "NO", req: false, inc: NEVER}
	keyProto           = configKey{name: "proto", def: "https", req: false, inc: NEVER}

	keysAll = []configKey{keyProtocolVersion, keyURL, keyUsername, keyToken, keyHostname, keyTld,
		keyMyIP, keyMx, keyBackMx, keyWildcard, keyInterval}
)

// AppConfig provides convenience methods for fetching ShadowCrypt
// specific properties.
type AppConfig struct {
	cfg.Config
	verified bool // ensures factory method must be used
}

// NewAppConfig creates an instance of AppConfig and verifies the
// contents of the specified config file.
func NewAppConfig(file string) (*AppConfig, error) {
	config := &AppConfig{verified: false}

	// create file Source using file spec and append
	// to Config
	src, err := cfg.NewSrcFileFromFilespec(file)
	if err != nil {
		return config, err
	}
	config.AppendSource(src)

	// Verify all the required properties exist.
	err = config.verify()
	if err == nil {
		config.verified = true
	}
	return config, err
}

// NewAppConfigFromMap creates an instance of AppConfig containing
// a copy of the specified map elements.
func NewAppConfigFromMap(m map[string]string) (*AppConfig, error) {
	config := &AppConfig{verified: false}
	src := cfg.NewSrcMapFromMap(m)
	config.AppendSource(src)
	err := config.verify()
	if err == nil {
		config.verified = true
	}
	return config, err
}

// getKeyVal returns the value of the specified key.
func (config *AppConfig) getKeyVal(key configKey) string {
	val, _ := config.String(key.name, key.def)
	return val
}

// Verify all the required properties exist
func (config *AppConfig) verify() error {
	// Check all required keys are present with non-empty values
	for _, k := range keysAll {
		if k.req {
			val, err := config.String(k.name, "")
			if err != nil || val == "" {
				return fmt.Errorf("key %s missing", k.name)
			}
		}
	}

	// Append another Source containing the defaults for all keys.
	m := make(map[string]string)
	for _, k := range keysAll {
		m[k.name] = k.def
	}
	config.AppendSource(cfg.NewSrcMapFromMap(m))
	return nil
}

// getKeys returns a slice containing all config keys.
func (config *AppConfig) getKeys() []configKey {
	return keysAll
}

func isFalse(s string) bool {
	s = strings.ToUpper(s)
	return s == "NO" || s == "OFF" || s == "FALSE"
}

func isTrue(s string) bool {
	s = strings.ToUpper(s)
	return s == "YES" || s == "ON" || s == "TRUE"
}

// Dump returns a string containing all application config properties.
func (config *AppConfig) Dump() string {
	var sb strings.Builder
	sep := ""
	for _, k := range keysAll {
		sb.WriteString(sep)
		sb.WriteString(k.name)
		sb.WriteString("=")
		val, _ := config.String(k.name, "<missing>")
		if val == "" {
			sb.WriteString("\"\"")
		} else {
			sb.WriteString(val)
		}
		sep = ", "
	}
	return sb.String()
}
