package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	defaultAppview = "https://bobbin.klbr.net"
	configName     = "config"
	configType     = "toml"
)

// config resolves values with the following precedence (highest to lowest):
// command-line flags, environment variables prefixed TG_, config file, defaults.
type settings struct {
	Appview string
	Account string
}

type flagSettings struct {
	ConfigPath string
	Appview    string
	Account    string
	ConfigSet  bool
	AppviewSet bool
	AccountSet bool
}

func loadConfig(flags flagSettings, errorWriter io.Writer) settings {
	config := viper.NewWithOptions(viper.KeyDelimiter("."))
	configPath := flags.ConfigPath
	if !flags.ConfigSet && configPath == "" {
		configPath = os.Getenv("TG_CONFIG")
	}
	config.SetConfigName(configName)
	config.SetConfigType(configType)

	if configPath != "" {
		config.SetConfigFile(configPath)
	} else {
		for _, dir := range configSearchDirs() {
			config.AddConfigPath(dir)
		}
	}

	config.SetEnvPrefix("TG")
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	config.AutomaticEnv()
	config.SetDefault("appview", defaultAppview)
	config.SetDefault("account", "")

	if err := config.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			// A missing config file is fine; configuration is optional.
			return applyFlagSettings(settings{
				Appview: config.GetString("appview"),
				Account: config.GetString("account"),
			}, flags)
		}
		// Surface parse/permission errors but keep running with defaults.
		fmt.Fprintln(errorWriter, "warning: failed to read config:", err)
	}
	resolved := settings{Appview: config.GetString("appview"), Account: config.GetString("account")}
	return applyFlagSettings(resolved, flags)
}

func applyFlagSettings(resolved settings, flags flagSettings) settings {
	if flags.AppviewSet {
		resolved.Appview = flags.Appview
	}
	if flags.AccountSet {
		resolved.Account = flags.Account
	}
	return resolved
}

func configSearchDirs() []string {
	var dirs []string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		dirs = append(dirs, filepath.Join(xdg, "tg"))
	} else if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".config", "tg"))
	}
	dirs = append(dirs, ".")
	return dirs
}
