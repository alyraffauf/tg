package cli

import (
	"errors"
	"fmt"
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
	Appview  string
	Account  string
	Knot     string
	SSHPort  string
	Protocol string
}

type flagSettings struct {
	ConfigPath string
	Appview    string
	Account    string
	ConfigSet  bool
	AppviewSet bool
	AccountSet bool
}

func loadConfig(flags flagSettings) (settings, error) {
	config := viper.NewWithOptions(viper.KeyDelimiter("."))
	configPath := flags.ConfigPath
	explicitConfig := flags.ConfigSet
	if !explicitConfig && configPath == "" {
		configPath = os.Getenv("TG_CONFIG")
		explicitConfig = configPath != ""
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
	config.SetDefault("knot", "")
	config.SetDefault("ssh-port", "22")
	config.SetDefault("protocol", "ssh")

	if err := config.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			if explicitConfig {
				return settings{}, fmt.Errorf("read config %q: %w", configPath, err)
			}
			return applyFlagSettings(settings{
				Appview:  config.GetString("appview"),
				Account:  config.GetString("account"),
				Knot:     config.GetString("knot"),
				SSHPort:  config.GetString("ssh-port"),
				Protocol: config.GetString("protocol"),
			}, flags), nil
		}
		return settings{}, fmt.Errorf("read config: %w", err)
	}
	resolved := settings{
		Appview:  config.GetString("appview"),
		Account:  config.GetString("account"),
		Knot:     config.GetString("knot"),
		SSHPort:  config.GetString("ssh-port"),
		Protocol: config.GetString("protocol"),
	}
	return applyFlagSettings(resolved, flags), nil
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
	return dirs
}
