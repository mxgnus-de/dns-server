package config

import (
	"fmt"
	"net"
	"os"
	"simpledns/internal/logger"

	"github.com/miekg/dns"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

type RecordType string

type Record struct {
	Name  string     `yaml:"name"`
	Type  RecordType `yaml:"type"`
	Value string     `yaml:"value"`
	TTL   uint32     `yaml:"ttl"`
	Class dns.Class  `yaml:"class"`
}

type Config struct {
	Records []*Record `yaml:"records"`
}

const (
	A     RecordType = "A"
	AAAA  RecordType = "AAAA"
	CNAME RecordType = "CNAME"
)

var (
	serviceLogger zerolog.Logger

	Cfg *Config
)

func Init(path string) {
	serviceLogger = logger.CreateService("config")
	Cfg = New(path)
	if err := Cfg.Validate(); err != nil {
		serviceLogger.Fatal().Err(err).Msg("Failed to validate config")
	}

	serviceLogger.Info().Int("records", len(Cfg.Records)).Msg("Config loaded")
}

func New(path string) *Config {
	cfg := &Config{}
	file, err := os.Open(path)
	if err != nil {
		serviceLogger.Fatal().Err(err).Str("path", path).Msg("Failed to open config file")
	}

	if err := yaml.NewDecoder(file).Decode(cfg); err != nil {
		serviceLogger.Fatal().Err(err).Str("path", path).Msg("Failed to decode config file")
	}

	cfg.applyDefaults()
	return cfg
}

func (cfg *Config) Validate() error {
	if len(cfg.Records) == 0 {
		return fmt.Errorf("no records defined")
	}

	for i, record := range cfg.Records {
		if record.Class != dns.ClassINET && record.Class != dns.ClassCSNET && record.Class != dns.ClassCHAOS && record.Class != dns.ClassHESIOD && record.Class != dns.ClassNONE && record.Class != dns.ClassANY {
			return fmt.Errorf("invalid class: %s", record.Class)
		}

		switch record.Type {
		case A:
			if net.ParseIP(record.Value) == nil {
				return fmt.Errorf("invalid IPv4 address: %s", record.Value)
			}

			if record.Name[len(record.Name)-1] != '.' {
				cfg.Records[i].Name = record.Name + "." // add trailing dot
			}
		case AAAA:
			if net.ParseIP(record.Value) == nil {
				return fmt.Errorf("invalid IPv6 address: %s", record.Value)
			}

			if record.Name[len(record.Name)-1] != '.' {
				cfg.Records[i].Name = record.Name + "." // add trailing dot
			}
		case CNAME:
			if record.Value == "" {
				return fmt.Errorf("invalid CNAME value: %s", record.Value)
			}

			if record.Name[len(record.Name)-1] != '.' {
				cfg.Records[i].Name = record.Name + "." // add trailing dot
			}

			if record.Value[len(record.Value)-1] != '.' {
				cfg.Records[i].Value = record.Value + "." // add trailing dot
			}
		default:
			return fmt.Errorf("invalid record type: %s", record.Type)
		}
	}
	return nil
}

func (cfg *Config) applyDefaults() {
	for _, record := range cfg.Records {
		if record.TTL == 0 {
			record.TTL = 3600
		}
		if record.Class == 0 {
			record.Class = dns.ClassINET
		}
	}
}
