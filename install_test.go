package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestServiceCmdLineArgs(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	absConfig := filepath.Join(t.TempDir(), "dynip.conf")

	tests := []struct {
		name   string
		config string
		want   []string
	}{
		{name: "no config", config: "", want: []string{"-d"}},
		{name: "absolute config", config: absConfig, want: []string{"-d", "-f", absConfig}},
		{name: "relative config", config: "dynip.conf", want: []string{"-d", "-f", filepath.Join(wd, "dynip.conf")}},
		{name: "relative subdir config", config: filepath.Join("conf", "..", "conf", "dynip.conf"),
			want: []string{"-d", "-f", filepath.Join(wd, "conf", "dynip.conf")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := serviceCmdLineArgs(tt.config)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, tt.want) {
				t.Fatalf("serviceCmdLineArgs() = %v, want %v", got, tt.want)
			}
			opts, _, err := parseFlags("dynip", got, nil)
			if err != nil {
				t.Fatalf("installed service args do not parse: %v", err)
			}
			if !opts.daemon {
				t.Error("installed service args do not select daemon mode")
			}
			if tt.config != "" && opts.config != tt.want[2] {
				t.Errorf("config = %q, want %q", opts.config, tt.want[2])
			}
		})
	}
}

func TestServiceConfig(t *testing.T) {
	cfg := serviceConfig()
	if cfg.Name != "dynip" {
		t.Errorf("Name = %q, want dynip", cfg.Name)
	}
	if cfg.Option == nil {
		t.Error("Option map is nil; serviceInstall writes Password into it")
	}
}
