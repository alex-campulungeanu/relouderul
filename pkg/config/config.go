package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/alex-campulungeanu/relouderul/pkg/helper"
)

type Runner interface {
	Run(cmd *exec.Cmd) error
}

// Implementations
type OSPathProvider struct {
	HomeDirFunc func() (string, error)
}

type OSRunner struct{}

type FileStore struct {
	PathProvider OSPathProvider
}

type OSEditor struct {
	PathProvider OSPathProvider
	Runner       Runner
}

type Service struct {
	Store  FileStore
	Editor OSEditor
}

func (p OSPathProvider) GetConfigPath() (string, error) {
	homeDir, err := helper.HomeDir()
	if err != nil {
		return "", fmt.Errorf("while trying to fetch config from home dir %w", err)
	}
	configFilePath := filepath.Join(homeDir, ".config", configFileDir, configFileName)
	slog.Info("Config file path", "path", configFilePath)
	return configFilePath, nil
}

func (r OSRunner) Run(cmd *exec.Cmd) error {
	return cmd.Run()
}

func (fs FileStore) Create() (string, error) {
	configFilePath, err := fs.PathProvider.GetConfigPath()
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(filepath.Dir(configFilePath), 0755)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(configFilePath); errors.Is(err, os.ErrNotExist) {
		file, err := os.Create(configFilePath)
		if err != nil {
			return "", err
		}
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(configTemplate); err != nil {
			return "", err
		}
		defer file.Close()
	}

	return configFilePath, nil
}

func (fs FileStore) Read() (ConfigStructure, error) {
	configFilePath, err := fs.PathProvider.GetConfigPath()
	if err != nil {
		return ConfigStructure{}, err
	}
	rawData, err := os.ReadFile(configFilePath)
	configData := ConfigStructure{}
	if err := json.Unmarshal(rawData, &configData); err != nil {
		return ConfigStructure{}, err
	}
	if err != nil {
		return ConfigStructure{}, err
	}
	return configData, nil
}

func (e OSEditor) Edit() error {
	configFilePath, err := e.PathProvider.GetConfigPath()
	if err != nil {
		return fmt.Errorf("unable to get the config path %w", err)
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf("no editor found")
	}
	slog.Info("Trying to edit the config file using", "editor", editor, "configFilePath", configFilePath)
	cmd := exec.Command(editor, configFilePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return e.Runner.Run(cmd)
}

func (s *Service) Create() (string, error) {
	return s.Store.Create()
}

func (s *Service) Read() (ConfigStructure, error) {
	return s.Store.Read()
}

func (s *Service) Edit() error {
	return s.Editor.Edit()
}

func NewOSPathProvider() OSPathProvider {
	return OSPathProvider{
		HomeDirFunc: helper.HomeDir,
	}
}

func Init(s Service) {
	filePath, err := s.Store.Create()
	if err != nil {
		slog.Error("error create config file %v", "err", err)
	}

	configFile, err := os.Open(filePath)
	if err != nil {
		slog.Error("error open config file %v", "err", err)
	}
	defer configFile.Close()
	byteValue, _ := io.ReadAll(configFile)
	var Config ConfigStructure
	if err := json.Unmarshal(byteValue, &Config); err != nil {
		slog.Error("error unmarshal config file %v", "err", err)
	}

}
