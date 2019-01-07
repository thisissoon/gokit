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
	}
	type Config struct {
		Log Log
	}

	// example env var override
	os.Setenv("TEST_LOG_LEVEL", "error")

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
		t.Errorf("unexpected value for Log.Verbose; expected %t, got %t", true, c.Log.Verbose)
	}
	if !c.Log.Console {
		t.Errorf("unexpected value for Log.Console; expected %t, got %t", true, c.Log.Console)
	}
	if c.Log.Level != "error" {
		t.Errorf("unexpected value for Log.Level; expected %s, got %s", "error", c.Log.Level)
	}
	if c.Log.Name != "blah" {
		t.Errorf("unexpected value for Log.Name; expected %s, got %s", "blah", c.Log.Name)
	}
}
