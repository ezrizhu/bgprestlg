package config

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
)

var (
	k      = koanf.New(".")
	parser = toml.Parser()

	Config struct {
		API struct {
			Host        string `koanf:"host"`
			Port        int    `koanf:"port"`
			BehindProxy bool   `koanf:"behind_proxy"`
		} `koanf:"api"`

		BGP struct {
			Host string `koanf:"host"`
		} `koanf:"bgp"`

		Peer struct {
			Address string `koanf:"address"`
			Port    int    `koanf:"port"`
			ASN     int    `koanf:"asn"`
		} `koanf:"peer"`
	}
)

func Load(configPath string) (err error) {
	// Load from toml
	if err := k.Load(file.Provider(configPath), toml.Parser()); err != nil {
		return err
	}

	// Marshal into struct
	if err := k.Unmarshal("", &Config); err != nil {
		return err
	}

	return
}
