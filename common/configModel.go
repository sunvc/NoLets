package common

import (
	"fmt"
	"os"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	System  System  `mapstructure:"system" json:"system" yaml:"system" koanf:"system"`
	Apple   Apple   `mapstructure:"apple" json:"apple" yaml:"apple" koanf:"apple"`
	Harmony Harmony `mapstructure:"harmony" json:"harmony" yaml:"harmony"`
}

type System struct {
	User                  string        `mapstructure:"user" json:"user" yaml:"user" koanf:"user"`
	Password              string        `mapstructure:"password" json:"password" yaml:"password" koanf:"password"`
	PushPassword          string        `mapstructure:"push_password" yaml:"push_password" koanf:"push_password"`
	SignKey               string        `mapstructure:"sign_key" json:"sign_key" yaml:"sign_key" koanf:"sign_key"`
	Addr                  string        `mapstructure:"addr" json:"addr" yaml:"addr" koanf:"addr"`
	URLPrefix             string        `mapstructure:"url_prefix" json:"url_prefix" yaml:"url_prefix" koanf:"url_prefix"`
	DataDir               string        `mapstructure:"data" json:"data" yaml:"data" koanf:"data"`
	DSN                   string        `mapstructure:"dsn" json:"dsn" yaml:"dsn" koanf:"dsn"`
	Cert                  string        `mapstructure:"cert" json:"cert" yaml:"cert" koanf:"cert"`
	Key                   string        `mapstructure:"key" json:"key" yaml:"key" koanf:"key"`
	ReduceMemoryUsage     bool          `mapstructure:"reduce_memory_usage" json:"reduce_memory_usage" yaml:"reduce_memory_usage" koanf:"reduce_memory_usage"`
	Voice                 bool          `mapstructure:"voice" json:"voice" yaml:"voice" koanf:"voice"`
	ProxyHeader           string        `mapstructure:"proxy_header" json:"proxy_header" yaml:"proxy_header" koanf:"proxy_header"`
	MaxBatchPushCount     int           `mapstructure:"max_batch_push_count" json:"max_batch_push_count" yaml:"max_batch_push_count" koanf:"max_batch_push_count"`
	MaxAPNSClientCount    int           `mapstructure:"max_apns_client_count" json:"max_apns_client_count" yaml:"max_apns_client_count" koanf:"max_apns_client_count"`
	MaxDeviceKeyArrLength int           `mapstructure:"max_device_key_arr_length" json:"max_device_key_arr_length" yaml:"max_device_key_arr_length" koanf:"max_device_key_arr_length"`
	Concurrency           int           `mapstructure:"concurrency" json:"concurrency" yaml:"concurrency" koanf:"concurrency"`
	ReadTimeout           time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout" koanf:"read_timeout"`
	WriteTimeout          time.Duration `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout" koanf:"write_timeout"`
	IdleTimeout           time.Duration `mapstructure:"idle_timeout" json:"idle_timeout" yaml:"idle_timeout" koanf:"idle_timeout"`
	Debug                 bool          `mapstructure:"debug" json:"debug" yaml:"debug" koanf:"debug"`
	Version               string        `mapstructure:"-" json:"-" yaml:"-" koanf:"-"`
	BuildDate             string        `mapstructure:"-" json:"-" yaml:"-" koanf:"-"`
	CommitID              string        `mapstructure:"-" json:"-" yaml:"-" koanf:"-"`
	CustomHttps           bool          `mapstructure:"-" json:"-" yaml:"-" koanf:"-"`
	ProxyDownload         bool          `mapstructure:"proxyDownload" json:"proxyDownload" yaml:"proxyDownload" koanf:"proxyDownload"`
	HideHome              bool          `mapstructure:"hideHome" json:"hideHome" yaml:"hideHome" koanf:"hideHome"`
	LogPath               string        `mapstructure:"logPath" json:"logPath" yaml:"logPath" koanf:"logPath"`
	ICPInfo               string        `mapstructure:"icp_info" json:"icp_info" yaml:"icp_info" koanf:"icp_info"`
	TimeZone              string        `mapstructure:"time_zone" json:"time_zone" yaml:"time_zone" koanf:"time_zone"`
	Auths                 []string      `mapstructure:"auths" json:"auths" yaml:"auths" koanf:"auths"`
}

type Apple struct {
	ApnsPrivateKey string `mapstructure:"apnsPrivateKey" json:"apnsPrivateKey" yaml:"apnsPrivateKey" koanf:"apnsPrivateKey"`
	Topic          string `mapstructure:"topic" json:"topic" yaml:"topic" koanf:"topic"`
	KeyID          string `mapstructure:"keyID" json:"keyID" yaml:"keyID" koanf:"keyID"`
	TeamID         string `mapstructure:"teamID" json:"teamID" yaml:"teamID" koanf:"teamID"`
	Develop        bool   `mapstructure:"develop" json:"develop" yaml:"develop" koanf:"develop"`
}

type Harmony struct {
	ProjectID           string `mapstructure:"project_id" json:"project_id" yaml:"project_id" koanf:"project_id"`
	KeyID               string `mapstructure:"key_id" json:"key_id" yaml:"key_id" koanf:"key_id"`
	PrivateKey          string `mapstructure:"private_key" json:"private_key" yaml:"private_key" koanf:"private_key"`
	SubAccount          string `mapstructure:"sub_account" json:"sub_account" yaml:"sub_account" koanf:"sub_account"`
	AuthURI             string `mapstructure:"auth_uri" json:"auth_uri" yaml:"auth_uri" koanf:"auth_uri"`
	TokenURI            string `mapstructure:"token_uri" json:"token_uri" yaml:"token_uri" koanf:"token_uri"`
	AuthProviderCertURI string `mapstructure:"auth_provider_cert_uri" json:"auth_provider_cert_uri" yaml:"auth_provider_cert_uri" koanf:"auth_provider_cert_uri"`
	ClientCertURI       string `mapstructure:"client_cert_uri" json:"client_cert_uri" yaml:"client_cert_uri" koanf:"client_cert_uri"`
	ClientId            string `mapstructure:"client_id" json:"client_id" yaml:"client_id" koanf:"client_id"`
}

func (global *Config) SetConfig(configPath string) error {
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config file %q not accessible: %w", configPath, err)
	}

	ko := koanf.New(".")
	if err := ko.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		return fmt.Errorf("load config %q: %w", configPath, err)
	}

	if err := ko.UnmarshalWithConf("", global, koanf.UnmarshalConf{
		Tag: "koanf",
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.StringToSliceHookFunc(","),
			),
		},
	}); err != nil {
		return fmt.Errorf("parse config %q: %w", configPath, err)
	}

	return nil
}
