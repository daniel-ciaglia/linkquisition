package freedesktop

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/strobotti/linkquisition"
)

var configDirPerms os.FileMode = 0o700
var configFilePerms os.FileMode = 0o600

var _ linkquisition.SettingsService = (*SettingsService)(nil)

type SettingsService struct {
	BrowserService linkquisition.BrowserService
}

func (s *SettingsService) GetConfigFilePath() string {
	return filepath.Join(s.GetConfigFolderPath(), "config.json")
}

func (s *SettingsService) GetConfigFolderPath() string {
	// get the user's home directory
	configDir, err := os.UserConfigDir()
	if err != nil {
		return ".config"
	}

	return filepath.Join(configDir, "linkquisition")
}

func (s *SettingsService) GetLogFilePath() string {
	return filepath.Join(s.GetLogFolderPath(), "linkquisition.log")
}

func (s *SettingsService) GetLogFolderPath() string {
	stateDir, isset := os.LookupEnv("XDG_STATE_HOME")
	if isset {
		return filepath.Join(stateDir, "linkquisition")
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		return filepath.Join(homeDir, ".local", "state", "linkquisition")
	}

	return filepath.Join(os.TempDir(), "linkquisition")
}

func (s *SettingsService) GetPluginFolderPaths() []string {
	var paths []string

	// XDG_DATA_HOME (default: ~/.local/share)
	dataHome, isset := os.LookupEnv("XDG_DATA_HOME")
	if !isset {
		if home, err := os.UserHomeDir(); err == nil {
			dataHome = filepath.Join(home, ".local", "share")
		}
	}
	if dataHome != "" {
		paths = append(paths, filepath.Join(dataHome, "linkquisition", "plugins"))
	}

	// XDG_DATA_DIRS (default: /usr/local/share:/usr/share)
	dataDirs, isset := os.LookupEnv("XDG_DATA_DIRS")
	if !isset {
		dataDirs = "/usr/local/share:/usr/share"
	}
	for dir := range strings.SplitSeq(dataDirs, ":") {
		if dir != "" {
			paths = append(paths, filepath.Join(dir, "linkquisition", "plugins"))
		}
	}

	return paths
}

func (s *SettingsService) ReadSettings() (*linkquisition.Settings, error) {
	data, err := os.ReadFile(s.GetConfigFilePath())
	if err != nil {
		return nil, fmt.Errorf("unable to open config-file `%s` for reading: %v", s.GetConfigFilePath(), err)
	}

	var settings = &linkquisition.Settings{}

	if err := json.Unmarshal(data, settings); err != nil {
		return nil, fmt.Errorf("unable to parse the config-file `%s`: %v", s.GetConfigFilePath(), err)
	}

	return settings, nil
}

func (s *SettingsService) WriteSettings(settings *linkquisition.Settings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("unable to marshal settings: %v", err)
	}

	// ensure the directory exists
	if errMkdir := os.MkdirAll(s.GetConfigFolderPath(), configDirPerms); errMkdir != nil {
		return fmt.Errorf("failed to write settings: %v", errMkdir)
	}

	if errWrite := os.WriteFile(s.GetConfigFilePath(), data, configFilePerms); errWrite != nil {
		return fmt.Errorf("failed to write settings: %v", errWrite)
	}

	return nil
}

func (s *SettingsService) IsConfigured() (bool, error) {
	if _, err := os.Stat(s.GetConfigFilePath()); errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	_, err := s.ReadSettings()

	return err == nil, err
}

func (s *SettingsService) GetSettings() *linkquisition.Settings {
	isConfigured, err := s.IsConfigured()
	if !isConfigured || err != nil {
		return linkquisition.GetDefaultSettings()
	}

	settings, err := s.ReadSettings()
	if err != nil {
		return linkquisition.GetDefaultSettings()
	}

	return settings
}

func (s *SettingsService) ScanBrowsers() error {
	var oldSettings *linkquisition.Settings

	if isConfigured, configErr := s.IsConfigured(); !isConfigured || configErr != nil {
		oldSettings = &linkquisition.Settings{}
	} else {
		var err error
		if oldSettings, err = s.ReadSettings(); err != nil {
			return fmt.Errorf("failed to scan browsers: %v", err)
		}
	}

	browsers, err := s.BrowserService.GetAvailableBrowsers()
	if err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	newSettings := oldSettings.UpdateWithBrowsers(browsers).NormalizeBrowsers()

	// ensure the directory exists
	if errMkdir := os.MkdirAll(s.GetConfigFolderPath(), configDirPerms); errMkdir != nil {
		return fmt.Errorf("failed to scan browsers: %v", errMkdir)
	}

	data, err := json.MarshalIndent(newSettings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	//nolint:mnd
	if err := os.WriteFile(s.GetConfigFilePath(), data, 0600); err != nil {
		return fmt.Errorf("failed to scan browsers: %v", err)
	}

	return nil
}
