package config

import (
	"fmt"
	"io/ioutil"
	"net/url"

	"gopkg.in/yaml.v2"
)

type ApiGatewayConfig struct {
	Server BindAddressConfig `yaml:"server"`
	Routes []RouteConfig     `yaml:"routes"`
}

type BindAddressConfig struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

type RouteConfig struct {
	FrontendConfig `yaml:"frontend"`
	BackendConfig  `yaml:"backend"`
}

type FrontendConfig struct {
	Methods []string `yaml:"methods"`
	Path    string   `yaml:"path"`
}

type BackendConfig struct {
	Url string `yaml:"url"`
}

func (b *BindAddressConfig) GetListenAddress() string {
	return fmt.Sprintf("%s:%d", b.Address, b.Port)
}

func (b *BackendConfig) GetUrl() (*url.URL, error) {
	backendUrl, err := url.ParseRequestURI(b.Url)
	if err != nil {
		return nil, err
	}
	return backendUrl, nil
}

func LoadConfig(filePath string) (*ApiGatewayConfig, error) {
	cfg := &ApiGatewayConfig{}

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
