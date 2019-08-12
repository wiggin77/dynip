package main

import (
	"fmt"

	"github.com/wiggin77/cfg"
)

type configKey struct {
	name string
	def  string
	req  bool
}

// Configuration keys
var (
	keyProtocolVersion = configKey{name: "protocol_ver", def: "1.3", req: false}
	keyURL             = configKey{name: "url", def: "api.cp.easydns.com/dyn/generic.php", req: false}
	keyUsername        = configKey{name: "username", def: "", req: true}
	keyToken           = configKey{name: "token", def: "", req: true}
	keyHostname        = configKey{name: "hostname", def: "", req: true}
	keyTld             = configKey{name: "tld", def: "", req: false}
	keyMyIP            = configKey{name: "myip", def: "1.1.1.1", req: false}
	keyMx              = configKey{name: "mx", def: "", req: false}
	keyBackMx          = configKey{name: "backmx", def: "NO", req: false}
	keyWildcard        = configKey{name: "wildcard", def: "OFF", req: false}

	keys = []configKey{keyProtocolVersion, keyURL, keyUsername, keyToken, keyHostname, keyTld, keyMyIP, keyMx, keyBackMx, keyWildcard}
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

// ShadowDir returns the shadow directory filespec
/*
func (config *AppConfig) ShadowDir() string {
	sdir, err := config.String(AbsKeyShadowDir, "")
	if err != nil {
		panic(err)
	}
	return sdir

}
*/

// Verify all the required properties exist
func (config *AppConfig) verify() error {
	// Check all required keys are present with non-empty values
	for _, k := range keys {
		if k.req {
			val, err := config.String(k.name, "")
			if err != nil || val == "" {
				return fmt.Errorf("key %s missing", k.name)
			}
		}
	}

	// Append another Source containing the defaults for all keys.
	m := make(map[string]string)
	for _, k := range keys {
		m[k.name] = k.def
	}
	config.AppendSource(cfg.NewSrcMapFromMap(m))
	return nil
}
