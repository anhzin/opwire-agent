package config

import (
	"encoding/json"
	"log"
	"github.com/opwire/opwire-agent/invokers"
	"github.com/opwire/opwire-agent/storages"
)

type Configuration struct {
	Version string `json:"version"`
	Main *invokers.CommandEntrypoint `json:"main-resource"`
	Resources map[string]invokers.CommandEntrypoint `json:"resources"`
	Unformed map[string]interface{} `json:"unformed"`
}

type Manager struct {
	currentVersion string
	defaultCfgFile string
	locator *Locator
	validator *Validator
}

func NewManager(currentVersion string, defaultCfgFile string) (*Manager) {
	m := &Manager{}
	m.currentVersion = currentVersion
	m.defaultCfgFile = defaultCfgFile
	m.locator = NewLocator()
	m.validator = NewValidator()
	return m
}

func (m *Manager) Load() (cfg *Configuration, result ValidationResult, err error) {
	cfg, err = m.loadJson()
	if cfg == nil || err != nil {
		return nil, nil, err
	}
	result, err = m.validator.Validate(cfg)
	return cfg, result, err
}

func (m *Manager) loadJson() (*Configuration, error) {
	fs := storages.GetFs()
	cfgpath, from := m.locator.GetConfigPath(m.defaultCfgFile)
	if len(from) == 0 {
		log.Printf("Configuration file not found")
		return nil, nil
	} else {
		log.Printf("Configuration path [%s] from [%s]", cfgpath, from)
	}

	config := &Configuration{}
	configFile, err := fs.Open(cfgpath)
	defer configFile.Close()
	if err != nil {
		return nil, err
	}
	parser := json.NewDecoder(configFile)
	parser.Decode(config)
	return config, nil
}