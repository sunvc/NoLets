package common

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	"github.com/goccy/go-yaml"
)

// LocalConfig is the global configuration instance.
var LocalConfig = &Config{
	System: System{
		User:                  "",
		Password:              "",
		PushPassword:          "",
		SignKey:               "",
		Addr:                  "0.0.0.0:8080",
		URLPrefix:             "/",
		DataDir:               "./data",
		DSN:                   "",
		Cert:                  "",
		Key:                   "",
		ReduceMemoryUsage:     false,
		Voice:                 false,
		ProxyHeader:           "",
		MaxBatchPushCount:     -1,
		MaxAPNSClientCount:    1,
		MaxDeviceKeyArrLength: 10,
		Concurrency:           256 * 1024,
		ReadTimeout:           3 * time.Second,
		WriteTimeout:          3 * time.Second,
		IdleTimeout:           3 * time.Second,
		CustomHttps:           false,
		ProxyDownload:         false,
		Debug:                 false,
		Version:               "",
		BuildDate:             "",
		CommitID:              "",
		ICPInfo:               "",
		Auths:                 []string{},

		WSHeartbeatInterval: 15 * time.Second,
		WSReadTimeout:       60 * time.Second,
		WSRingBufferTTL:     5 * time.Second,
		WSSessionMaxHold:    90 * time.Second,
		WSSessionGCInterval: 5 * time.Second,
		WSMaxFrameBytes:     4096,
		WSSendQueueSize:     256,
	},
	Apple: Apple{
		ApnsPrivateKey: `-----BEGIN PRIVATE KEY-----
MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQgvjopbchDpzJNojnc
o7ErdZQFZM7Qxho6m61gqZuGVRigCgYIKoZIzj0DAQehRANCAAQ8ReU0fBNg+sA+
ZdDf3w+8FRQxFBKSD/Opt7n3tmtnmnl9Vrtw/nUXX4ldasxA2gErXR4YbEL9Z+uJ
REJP/5bp
-----END PRIVATE KEY-----`,
		Topic:   "me.uuneo.Meoworld",
		KeyID:   "BNY5GUGV38",
		TeamID:  "FUWV6U942Q",
		Develop: false,
	},
}

func SetDefaultVersionOrCommID(version, buildDate, commID string) {
	if len(version) > 0 {
		LocalConfig.System.Version = version
	} else {
		LocalConfig.System.Version = "v2.3.7"
	}
	if len(commID) > 0 {
		LocalConfig.System.CommitID = commID
	} else {
		LocalConfig.System.CommitID = "f7efb70"
	}
	if len(buildDate) > 0 {
		LocalConfig.System.BuildDate = buildDate
	} else {
		LocalConfig.System.BuildDate = "2025-01-01 09:20:33"
	}
}

// SynchronousFieldFile Prevent problems with the fields
func SynchronousFieldFile() {
	data, err := yaml.Marshal(LocalConfig)
	if err != nil {
		panic(err)
	}

	header := `# ============================================
# NoLet Server Configuration
# Generated automatically. Do not edit manually.
# Modify values carefully, then restart the service.
# ============================================

`
	finalData := append([]byte(header), data...)

	if err = os.WriteFile("config.yaml", finalData, 0644); err != nil {
		panic(err)
	}
	for _, item := range Flags() {
		fmt.Println(item.String())
	}
}
