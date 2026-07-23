package common

import (
	"log"
	"os"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	System System `mapstructure:"system" json:"system" yaml:"system" koanf:"system"`
	Apple  Apple  `mapstructure:"apple" json:"apple" yaml:"apple" koanf:"apple"`
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
	LogPath               string        `mapstructure:"logPath" json:"logPath" yaml:"logPath" koanf:"logPath"`
	ICPInfo               string        `mapstructure:"icp_info" json:"icp_info" yaml:"icp_info" koanf:"icp_info"`
	TimeZone              string        `mapstructure:"time_zone" json:"time_zone" yaml:"time_zone" koanf:"time_zone"`
	Auths                 []string      `mapstructure:"auths" json:"auths" yaml:"auths" koanf:"auths"`

	// WebSocket PTT settings
	WSHeartbeatInterval time.Duration `mapstructure:"ws_heartbeat_interval" json:"ws_heartbeat_interval" yaml:"ws_heartbeat_interval" koanf:"ws_heartbeat_interval"`
	WSReadTimeout       time.Duration `mapstructure:"ws_read_timeout" json:"ws_read_timeout" yaml:"ws_read_timeout" koanf:"ws_read_timeout"`
	WSRingBufferTTL     time.Duration `mapstructure:"ws_ring_buffer_ttl" json:"ws_ring_buffer_ttl" yaml:"ws_ring_buffer_ttl" koanf:"ws_ring_buffer_ttl"`
	WSSessionMaxHold    time.Duration `mapstructure:"ws_session_max_hold" json:"ws_session_max_hold" yaml:"ws_session_max_hold" koanf:"ws_session_max_hold"`
	WSSessionGCInterval time.Duration `mapstructure:"ws_session_gc_interval" json:"ws_session_gc_interval" yaml:"ws_session_gc_interval" koanf:"ws_session_gc_interval"`
	WSMaxFrameBytes     int           `mapstructure:"ws_max_frame_bytes" json:"ws_max_frame_bytes" yaml:"ws_max_frame_bytes" koanf:"ws_max_frame_bytes"`
	WSSendQueueSize     int           `mapstructure:"ws_send_queue_size" json:"ws_send_queue_size" yaml:"ws_send_queue_size" koanf:"ws_send_queue_size"`
}

type Apple struct {
	ApnsPrivateKey string `mapstructure:"apnsPrivateKey" json:"apnsPrivateKey" yaml:"apnsPrivateKey" koanf:"apnsPrivateKey"`
	Topic          string `mapstructure:"topic" json:"topic" yaml:"topic" koanf:"topic"`
	KeyID          string `mapstructure:"keyID" json:"keyID" yaml:"keyID" koanf:"keyID"`
	TeamID         string `mapstructure:"teamID" json:"teamID" yaml:"teamID" koanf:"teamID"`
	Develop        bool   `mapstructure:"develop" json:"develop" yaml:"develop" koanf:"develop"`
}

func (global *Config) SetConfig(configPath string) {

	var conf Config

	if _, err := os.Stat(configPath); err != nil {
		return
	}

	ko := koanf.New(".")
	// Load JSON common.
	if err := ko.Load(file.Provider(configPath), yaml.Parser()); err != nil {
		log.Fatalf("error loading common: %v", err)
		return
	}

	if err := ko.Unmarshal("", &conf); err != nil {
		log.Fatal(err)
		return
	}
	// check system params
	if len(conf.System.User) > 0 {
		global.System.User = conf.System.User
	}
	if len(conf.System.Password) > 0 {
		global.System.Password = conf.System.Password
	}
	if len(conf.System.Addr) > 0 {
		global.System.Addr = conf.System.Addr
	}
	if len(conf.System.URLPrefix) > 0 {
		global.System.URLPrefix = conf.System.URLPrefix
	}
	if len(conf.System.DataDir) > 0 {
		global.System.DataDir = conf.System.DataDir
	}

	if len(conf.System.DSN) > 0 {
		global.System.DSN = conf.System.DSN
	}

	if len(conf.System.Cert) > 0 {
		global.System.Cert = conf.System.Cert
	}
	if len(conf.System.Key) > 0 {
		global.System.Key = conf.System.Key
	}

	global.System.ReduceMemoryUsage = conf.System.ReduceMemoryUsage
	global.System.Voice = conf.System.Voice
	if len(conf.System.ProxyHeader) > 0 {
		global.System.ProxyHeader = conf.System.ProxyHeader
	}
	if conf.System.MaxBatchPushCount > 0 {
		global.System.MaxBatchPushCount = conf.System.MaxBatchPushCount
	}
	if conf.System.MaxAPNSClientCount > 0 {
		global.System.MaxAPNSClientCount = conf.System.MaxAPNSClientCount
	}
	if conf.System.Concurrency > 0 {
		global.System.Concurrency = conf.System.Concurrency
	}
	if conf.System.ReadTimeout > 0 {
		global.System.ReadTimeout = conf.System.ReadTimeout
	}
	if conf.System.WriteTimeout > 0 {
		global.System.WriteTimeout = conf.System.WriteTimeout
	}
	if conf.System.IdleTimeout > 0 {
		global.System.IdleTimeout = conf.System.IdleTimeout
	}
	global.System.Debug = conf.System.Debug
	if len(conf.System.Version) > 0 {
		global.System.Version = conf.System.Version
	}
	if len(conf.System.BuildDate) > 0 {
		global.System.BuildDate = conf.System.BuildDate
	}
	if len(conf.System.CommitID) > 0 {
		global.System.CommitID = conf.System.CommitID
	}

	if len(conf.System.ICPInfo) > 0 {
		global.System.ICPInfo = conf.System.ICPInfo
	}
	if len(conf.System.TimeZone) > 0 {
		global.System.TimeZone = conf.System.TimeZone
	}

	// WebSocket PTT overrides — only apply positive durations / sizes so the
	// caller's baked-in defaults survive an empty config.
	if conf.System.WSHeartbeatInterval > 0 {
		global.System.WSHeartbeatInterval = conf.System.WSHeartbeatInterval
	}
	if conf.System.WSReadTimeout > 0 {
		global.System.WSReadTimeout = conf.System.WSReadTimeout
	}
	if conf.System.WSRingBufferTTL > 0 {
		global.System.WSRingBufferTTL = conf.System.WSRingBufferTTL
	}
	if conf.System.WSSessionMaxHold > 0 {
		global.System.WSSessionMaxHold = conf.System.WSSessionMaxHold
	}
	if conf.System.WSSessionGCInterval > 0 {
		global.System.WSSessionGCInterval = conf.System.WSSessionGCInterval
	}
	if conf.System.WSMaxFrameBytes > 0 {
		global.System.WSMaxFrameBytes = conf.System.WSMaxFrameBytes
	}
	if conf.System.WSSendQueueSize > 0 {
		global.System.WSSendQueueSize = conf.System.WSSendQueueSize
	}

	if len(conf.Apple.ApnsPrivateKey) > 0 {
		global.Apple.ApnsPrivateKey = conf.Apple.ApnsPrivateKey
	}
	if len(conf.Apple.Topic) > 0 {
		global.Apple.Topic = conf.Apple.Topic
	}
	if len(conf.Apple.KeyID) > 0 {
		global.Apple.KeyID = conf.Apple.KeyID
	}
	if len(conf.Apple.TeamID) > 0 {
		global.Apple.TeamID = conf.Apple.TeamID
	}
	global.Apple.Develop = conf.Apple.Develop
}
