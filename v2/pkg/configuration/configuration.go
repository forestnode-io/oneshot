package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var userConfigPath string

type Configuration interface {
	Init()
	SetFlags(*cobra.Command, *viper.Viper)
	MergeFlags()
	Validate() error
}

func init() {
	if confPath := os.Getenv("ONESHOT_CONFIG"); confPath != "" {
		userConfigPath = confPath
		return
	}

	ucd, err := os.UserConfigDir()
	if err == nil {
		userConfigPath = filepath.Join(ucd, "oneshot")
	} else if errors.Is(err, os.ErrNotExist) {
		userConfigPath = "" // Don't use user config dir if no config dir exists
	} else {
		panic(fmt.Errorf("failed to get user config dir: %w", err))
	}

	_, err = os.Stat(userConfigPath)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(userConfigPath, 0700); err != nil {
			panic(fmt.Errorf("failed to create user config dir: %w", err))
		}
		// Dont use user config dir if it doesn't exist even if we just
		// created it since it guaranteed to be empty.
		userConfigPath = ""
	} else if err != nil {
		panic(fmt.Errorf("failed to stat user config dir: %w", err))
	}
}

func ReadConfig() (*Root, error) {
	var config Root
	if userConfigPath == "" {
		return &config, nil
	}

	data, err := os.ReadFile(userConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read user config file: %w", err)
	}

	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user config file: %w", err)
	}

	return &config, nil
}
