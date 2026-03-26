package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

// Config holds all runtime configuration for the application.
// Each field is populated from the environment variable named by the `env` tag.
// If the variable is absent, the value in `default_value` is used.
// Supported field types: string, int, bool.
type Config struct {
	Port      int    `env:"PORT"       default_value:"8080"`
	DBPath    string `env:"DB_PATH"    default_value:"./kpfc.db"`
	JWTSecret string `env:"JWT_SECRET" default_value:""`
}

// Load reads configuration from environment variables using struct reflection.
// Adding a new config field only requires adding a struct field with the appropriate tags.
func Load() (*Config, error) {
	cfg := &Config{}
	rv := reflect.ValueOf(cfg).Elem()
	rt := rv.Type()

	for i := range rt.NumField() {
		field := rt.Field(i)
		fv := rv.Field(i)

		envKey := field.Tag.Get("env")
		raw := os.Getenv(envKey)
		if raw == "" {
			raw = field.Tag.Get("default_value")
		}

		if err := setField(fv, field.Name, envKey, raw); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// setField converts raw (always a string from os.Getenv or the default tag)
// into the correct Go type and assigns it to fv.
func setField(fv reflect.Value, fieldName, envKey, raw string) error {
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if raw == "" {
			fv.SetInt(0)
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("config: %s (%s=%q) must be an integer: %w", fieldName, envKey, raw, err)
		}
		fv.SetInt(n)

	case reflect.Bool:
		if raw == "" {
			fv.SetBool(false)
			return nil
		}
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("config: %s (%s=%q) must be a boolean (true/false/1/0): %w", fieldName, envKey, raw, err)
		}
		fv.SetBool(b)

	default:
		return fmt.Errorf("config: unsupported field type %s for %s", fv.Kind(), fieldName)
	}

	return nil
}
