package config

import (
	"os"
	"testing"
)

type testConfig struct {
	DSN string `config:"dsn"`
}

func TestLoader(t *testing.T) {
	_ = os.Setenv("APP_DSN", "sqlite")
	defer os.Unsetenv("APP_DSN")

	var cfg testConfig
	if err := New("app").Load(&cfg); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.DSN != "sqlite" {
		t.Fatalf("DSN = %s", cfg.DSN)
	}
}
