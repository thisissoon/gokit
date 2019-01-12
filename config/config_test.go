package config_test

import (
	"os"
	"testing"

	"github.com/spf13/pflag"
	"go.soon.build/kit/config"
)

func TestConfig(t *testing.T) {
	type Log struct {
		Verbose bool
		Console bool
		Level   string
		Name    string
		Custom  string `mapstructure:"customTag"`
		Env     string `mapstructure:"envTest"`
	}
	type Config struct {
		Log Log
	}

	// example env var overrides
	os.Setenv("TEST_LOG_LEVEL", "error")
	os.Setenv("TEST_LOG_ENVTEST", "env")

	// example flag override
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	fs.String("name", "blah", "")

	// default config
	c := Config{
		Log: Log{
			Verbose: true,
		},
	}
	v := config.ViperWithDefaults("test")
	err := config.ReadInConfig(v, &c,
		config.WithFile("testdata/test.toml"),
		config.BindFlag("log.name", fs.Lookup("name")),
	)
	if err != nil {
		t.Error(err)
	}
	if !c.Log.Verbose {
		// value from default
		t.Errorf("unexpected value for Log.Verbose; expected %t, got %t", true, c.Log.Verbose)
	}
	if !c.Log.Console {
		// value from file by field name
		t.Errorf("unexpected value for Log.Console; expected %t, got %t", true, c.Log.Console)
	}
	if c.Log.Level != "error" {
		// value override with env var from field name `TEST_LOG_LEVEL`
		t.Errorf("unexpected value for Log.Level; expected %s, got %s", "error", c.Log.Level)
	}
	if c.Log.Name != "blah" {
		// value override with flag `--name`
		t.Errorf("unexpected value for Log.Name; expected %s, got %s", "blah", c.Log.Name)
	}
	if c.Log.Custom != "value" {
		// value from file by mapstructure tag
		t.Errorf("unexpected value for Log.Custom; expected %s, got %s", "value", c.Log.Custom)
	}
	if c.Log.Env != "env" {
		// value override with env var from mapstructure tag `TEST_LOG_ENVTEST`
		t.Errorf("unexpected value for Log.Env; expected %s, got %s", "env", c.Log.Env)
	}
}
