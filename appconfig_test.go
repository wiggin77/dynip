package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeConfigFile(t *testing.T, props map[string]string) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("# test config\n")
	for k, v := range props {
		sb.WriteString(k + " = " + v + "\n")
	}
	file := filepath.Join(t.TempDir(), "dynip.conf")
	if err := os.WriteFile(file, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func requiredProps() map[string]string {
	return map[string]string{
		"username": "testuser",
		"token":    "testtoken",
		"hostname": "test.example.com",
	}
}

func TestNewAppConfig(t *testing.T) {
	t.Run("valid file applies defaults", func(t *testing.T) {
		props := requiredProps()
		props["tld"] = ""
		props["wildcard"] = "ON"
		cfg, err := NewAppConfig(writeConfigFile(t, props))
		if err != nil {
			t.Fatal(err)
		}
		if got := cfg.getKeyVal(keyHostname); got != "test.example.com" {
			t.Errorf("hostname = %q", got)
		}
		if got := cfg.getKeyVal(keyWildcard); got != "ON" {
			t.Errorf("wildcard = %q, want ON", got)
		}
		if got := cfg.getKeyVal(keyMyIP); got != keyMyIP.def {
			t.Errorf("myip = %q, want default %q", got, keyMyIP.def)
		}
		if got := cfg.getKeyVal(keyURL); got != keyURL.def {
			t.Errorf("url = %q, want default %q", got, keyURL.def)
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := NewAppConfig(filepath.Join(t.TempDir(), "nope.conf"))
		if err == nil {
			t.Fatal("expected error for missing file")
		}
	})

	for _, key := range []string{"username", "token", "hostname"} {
		t.Run("missing "+key, func(t *testing.T) {
			props := requiredProps()
			delete(props, key)
			_, err := NewAppConfig(writeConfigFile(t, props))
			if err == nil || !strings.Contains(err.Error(), "key "+key+" missing") {
				t.Fatalf("err = %v, want key %s missing", err, key)
			}
		})

		t.Run("empty "+key, func(t *testing.T) {
			props := requiredProps()
			props[key] = ""
			_, err := NewAppConfig(writeConfigFile(t, props))
			if err == nil || !strings.Contains(err.Error(), "key "+key+" missing") {
				t.Fatalf("err = %v, want key %s missing", err, key)
			}
		})
	}
}

func TestNewAppConfigFromMapMissingKey(t *testing.T) {
	props := requiredProps()
	delete(props, "token")
	if _, err := NewAppConfigFromMap(props); err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestIsTrueIsFalse(t *testing.T) {
	tests := []struct {
		in        string
		wantTrue  bool
		wantFalse bool
	}{
		{"YES", true, false},
		{"yes", true, false},
		{"ON", true, false},
		{"on", true, false},
		{"TRUE", true, false},
		{"True", true, false},
		{"NO", false, true},
		{"no", false, true},
		{"OFF", false, true},
		{"FALSE", false, true},
		{"false", false, true},
		{"", false, false},
		{"maybe", false, false},
		{"1", false, false},
	}
	for _, tt := range tests {
		if got := isTrue(tt.in); got != tt.wantTrue {
			t.Errorf("isTrue(%q) = %v, want %v", tt.in, got, tt.wantTrue)
		}
		if got := isFalse(tt.in); got != tt.wantFalse {
			t.Errorf("isFalse(%q) = %v, want %v", tt.in, got, tt.wantFalse)
		}
	}
}

func TestDump(t *testing.T) {
	cfg, err := NewAppConfigFromMap(requiredProps())
	if err != nil {
		t.Fatal(err)
	}
	want := `protocol_ver=1.3, url=api.cp.easydns.com/dyn/generic.php, username=testuser, ` +
		`token=testtoken, hostname=test.example.com, tld="", myip=1.1.1.1, mx="", ` +
		`backmx=NO, wildcard=OFF, interval=11 minutes`
	if got := cfg.Dump(); got != want {
		t.Errorf("Dump() =\n%s\nwant\n%s", got, want)
	}
}
