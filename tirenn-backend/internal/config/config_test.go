package config

import (
	"os"
	"testing"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := LoadConfig()

	if cfg.Port == "" {
		t.Error("Expected default Port to be set, got empty")
	}
	if cfg.DBHost == "" {
		t.Error("Expected default DBHost to be set, got empty")
	}
	if cfg.JWTSecret == "" {
		t.Error("Expected default JWTSecret to be set, got empty")
	}
	if cfg.JWTExpireHours != 24 {
		t.Errorf("Expected default JWTExpireHours to be 24, got %d", cfg.JWTExpireHours)
	}
}

func TestLoadConfig_EnvOverride(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("DB_NAME", "custom_commerce_db")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DB_NAME")
	}()

	cfg := LoadConfig()

	if cfg.Port != "9090" {
		t.Errorf("Expected Port to be overridden to 9090, got %s", cfg.Port)
	}
	if cfg.DBName != "custom_commerce_db" {
		t.Errorf("Expected DBName to be overridden to custom_commerce_db, got %s", cfg.DBName)
	}
}
