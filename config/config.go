// Common configuration management with [viper](https://github.com/spf13/viper).
// Supports loading configuration from toml files, auto ENV var bindings and
// cobra command flag overrides.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// An Option function can provide extra viper configuration
type Option func(v *viper.Viper) error

// WithFile will override implicit configuration file lookups and specify an
// absolute path to a config file to load
func WithFile(p string) Option {
	return func(v *viper.Viper) error {
		if p != "" {
			v.SetConfigFile(p)
		}
		return nil
	}
}

// BindFlag returns an Option function allowing the binding of CLI flags to
// configuration values
func BindFlag(key string, flag *pflag.Flag) Option {
	return func(v *viper.Viper) error {
		if flag == nil {
			return nil
		}
		return v.BindPFlag(key, flag)
	}
}

// ViperWithDefaults constructs a new viper instance pre-configured with
// SOON_ defaults.
// - TOML format
// - Loads files from `/etc/name/name.toml`, `$HOME/.config/name.toml`
// - Env vars as `NAME_FIELD`
// - Names that are hyphenated will have the hyphens removed e.g. "new-project" would be accessed "NEWPROJECT"
func ViperWithDefaults(name string) *viper.Viper {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName(name)
	// Set default config paths
	v.AddConfigPath(fmt.Sprintf("/etc/%s", name))
	v.AddConfigPath("$HOME/.config")
	setEnv(v, name)
	return v
}

// ReadInConfig constructs a new Config instance
func ReadInConfig(v *viper.Viper, c interface{}, opts ...Option) error {
	for _, opt := range opts {
		err := opt(v)
		if err != nil {
			return err
		}
	}
	err := bindEnvs(v, c)
	if err != nil {
		return err
	}
	switch err := v.ReadInConfig(); err.(type) {
	case nil, viper.ConfigFileNotFoundError:
		break
	default:
		return err
	}
	return v.Unmarshal(c)
}

// ViperWithDir constructs a new viper instance pre-configured with
// SOON_ defaults which is used to load all config from a directory
// - TOML format
// - Loads files from `/etc/name/`
// - Env vars as `NAME_FIELD`
// - Names that are hyphenated will have the hyphens removed e.g. "new-project" would be accessed "NEWPROJECT"
func ViperWithDir(name string) (*viper.Viper, string) {
	v := viper.New()
	v.SetConfigType("toml")
	// Set default config paths
	path := fmt.Sprintf("/etc/%s", name)
	v.AddConfigPath(path)
	setEnv(v, name)
	return v, path
}

// ReadInAllDirConfig reads and merges all config files inside a directory
func ReadInAllDirConfig(v *viper.Viper, p string, c interface{}, opts ...Option) error {
	err := ReadInConfig(v, c, opts...)
	if err != nil {
		return err
	}
	if v.ConfigFileUsed() == "" {
		dir, err := os.ReadDir(p)
		if err != nil {
			return err
		}
		for i, d := range dir {
			fileName := strings.Split(d.Name(), ".")[0]
			if filepath.Ext(d.Name()) == ".toml" {
				v.SetConfigName(fileName)
				if i == 0 {
					err = ReadInConfig(v, c, opts...)
					if err != nil {
						return err
					}
				} else {
					err = v.MergeInConfig()
					if err != nil {
						return err
					}
				}
			}
		}
		return v.Unmarshal(c)
	}
	return nil
}

// lowerFirst lowercases the first character of a string
func lowerFirst(s string) string {
	a := []rune(s)
	a[0] = unicode.ToLower(a[0])
	return string(a)
}

// bindEnvs uses reflection to bind environment variables with viper.Unmarshal
// which cannot use viper.AutomaticEnv. It takes an interface which should be
// a struct or pointer to a struct and an optional slice of strings.
func bindEnvs(v *viper.Viper, iface interface{}, parts ...string) error {
	var ifv reflect.Value
	var ift reflect.Type
	if reflect.TypeOf(iface).Kind() == reflect.Ptr {
		ifv = reflect.Indirect(reflect.ValueOf(iface))
		ift = reflect.Indirect(reflect.ValueOf(iface)).Type()
	} else {
		ifv = reflect.ValueOf(iface)
		ift = reflect.TypeOf(iface)
	}
	for i := 0; i < ift.NumField(); i++ {
		val := ifv.Field(i)
		tv := ift.Field(i).Tag.Get("mapstructure")
		if tv == "" {
			// default to field name, with lowercased first char `Log` => log
			tv = lowerFirst(ift.Field(i).Name)
		}
		switch val.Kind() {
		case reflect.Struct:
			// If the field is a struct the name of the field is appended
			// to the parts slice and BindEnvs is called again with the nested
			// struct and the parts slice
			err := bindEnvs(v, val.Interface(), append(parts, tv)...)
			if err != nil {
				return err
			}
		default:
			// If it is not a struct the field name is joined with the parts
			// slice with `.` and viper.BindEnv is called with that name
			err := v.BindEnv(strings.Join(append(parts, tv), "."))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func setEnv(v *viper.Viper, name string) {
	envPrefix := strings.ReplaceAll(name, "-", "")
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}
