package common

import (
	"context"
	"time"

	"github.com/urfave/cli/v3"
)

// Flags returns the list of command-line flags supported by the application.
// These flags configure various aspects of the system, including server settings, database connection, APNs credentials, and more.
func Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "addr",
			Usage:       "Server listen address",
			Sources:     cli.EnvVars("NOLET_SERVER_ADDRESS"),
			Value:       "0.0.0.0:8080",
			Destination: &LocalConfig.System.Addr,
		},
		&cli.StringFlag{
			Name:        "url-prefix",
			Usage:       "Serve URL Prefix",
			Sources:     cli.EnvVars("NOLET_SERVER_URL_PREFIX"),
			Value:       "/",
			Destination: &LocalConfig.System.URLPrefix,
		},
		&cli.StringFlag{
			Name:        "dir",
			Usage:       "Server data storage dir",
			Sources:     cli.EnvVars("NOLET_SERVER_DATA_DIR"),
			Value:       "./data",
			Destination: &LocalConfig.System.DataDir,
		},
		&cli.StringFlag{
			Name:        "dsn",
			Usage:       "MySQL DSN user:pass@tcp(host)/dbname",
			Sources:     cli.EnvVars("NOLET_SERVER_DSN"),
			Destination: &LocalConfig.System.DSN,
		},
		&cli.StringFlag{
			Name:        "cert",
			Usage:       "Server TLS certificate",
			Sources:     cli.EnvVars("NOLET_SERVER_CERT"),
			Destination: &LocalConfig.System.Cert,
		},
		&cli.StringFlag{
			Name:        "key",
			Usage:       "Server TLS certificate key",
			Sources:     cli.EnvVars("NOLET_SERVER_KEY"),
			Destination: &LocalConfig.System.Key,
		},
		&cli.BoolFlag{
			Name:        "reduce-memory-usage",
			Usage:       "Aggressively reduces memory usage at the cost of higher CPU usage if set to true",
			Sources:     cli.EnvVars("NOLET_SERVER_REDUCE_MEMORY_USAGE"),
			Value:       false,
			Destination: &LocalConfig.System.ReduceMemoryUsage,
		},
		&cli.BoolFlag{
			Name:        "voice",
			Usage:       "Enable PushToTalk voice routes",
			Sources:     cli.EnvVars("NOLET_SERVER_VOICE"),
			Value:       false,
			Destination: &LocalConfig.System.Voice,
		},
		&cli.StringFlag{
			Name:        "user",
			Usage:       "Basic auth username",
			Sources:     cli.EnvVars("NOLET_SERVER_BASIC_AUTH_USER"),
			Aliases:     []string{"u"},
			Destination: &LocalConfig.System.User,
			Value:       "",
		},
		&cli.StringFlag{
			Name:        "password",
			Usage:       "Basic auth password",
			Sources:     cli.EnvVars("NOLET_SERVER_BASIC_AUTH_PASSWORD"),
			Aliases:     []string{"p"},
			Destination: &LocalConfig.System.Password,
			Value:       "",
		},
		&cli.StringFlag{
			Name:        "push-password",
			Usage:       "push auth password",
			Sources:     cli.EnvVars("NOLET_PUSH_PASSWORD"),
			Destination: &LocalConfig.System.PushPassword,
			Value:       "",
		},
		&cli.StringFlag{
			Name:        "sign-key",
			Usage:       "App Sign Key",
			Sources:     cli.EnvVars("NOLET_SIGN_KEY"),
			Aliases:     []string{"sk"},
			Destination: &LocalConfig.System.SignKey,
			Value:       "",
		},
		&cli.StringFlag{
			Name:        "proxy-header",
			Usage:       "The remote IP address used by the NOLET server http header",
			Sources:     cli.EnvVars("NOLET_SERVER_PROXY_HEADER"),
			Destination: &LocalConfig.System.ProxyHeader,
			Value:       "",
		},
		&cli.IntFlag{
			Name:        "max-batch-push-count",
			Usage:       "Maximum number of batch pushes allowed, -1 means no limit",
			Sources:     cli.EnvVars("NOLET_SERVER_MAX_BATCH_PUSH_COUNT"),
			Value:       -1,
			Destination: &LocalConfig.System.MaxBatchPushCount,
		},
		&cli.IntFlag{
			Name:        "max-apns-client-count",
			Usage:       "Maximum number of APNs client connections",
			Sources:     cli.EnvVars("NOLET_SERVER_MAX_APNS_CLIENT_COUNT"),
			Aliases:     []string{"max"},
			Value:       1,
			Destination: &LocalConfig.System.MaxAPNSClientCount,
		},
		&cli.IntFlag{
			Name:        "max-device-key-arr-length",
			Usage:       "Maximum number of deviceKey list length connections",
			Sources:     cli.EnvVars("NOLET_CONCURRENCY"),
			Value:       10,
			Destination: &LocalConfig.System.MaxDeviceKeyArrLength,
		},
		&cli.IntFlag{
			Name:        "concurrency",
			Usage:       "Maximum number of concurrent connections",
			Sources:     cli.EnvVars("NOLET_SERVER_CONCURRENCY"),
			Value:       256 * 1024,
			Hidden:      true,
			Destination: &LocalConfig.System.Concurrency,
		},
		&cli.DurationFlag{
			Name:        "read-timeout",
			Usage:       "The amount of time allowed to read the full request, including the body",
			Sources:     cli.EnvVars("NOLET_SERVER_READ_TIMEOUT"),
			Value:       3 * time.Second,
			Hidden:      true,
			Destination: &LocalConfig.System.ReadTimeout,
		},
		&cli.DurationFlag{
			Name:        "write-timeout",
			Usage:       "The maximum duration before timing out writes of the response",
			Sources:     cli.EnvVars("NOLET_SERVER_WRITE_TIMEOUT"),
			Value:       3 * time.Second,
			Hidden:      true,
			Destination: &LocalConfig.System.WriteTimeout,
		},
		&cli.DurationFlag{
			Name:        "idle-timeout",
			Usage:       "The maximum amount of time to wait for the next request when keep-alive is enabled",
			Sources:     cli.EnvVars("NOLET_SERVER_IDLE_TIMEOUT"),
			Value:       10 * time.Second,
			Hidden:      true,
			Destination: &LocalConfig.System.IdleTimeout,
		},
		&cli.BoolFlag{
			Name:        "debug",
			Value:       false,
			Usage:       "enable debug mode",
			Sources:     cli.EnvVars("NOLET_DEBUG"),
			Destination: &LocalConfig.System.Debug,
		},
		&cli.StringSliceFlag{
			Name:        "auths",
			Value:       []string{},
			Usage:       "auth id list",
			Sources:     cli.EnvVars("NOLET_AUTHS"),
			Destination: &LocalConfig.System.Auths,
		},
		&cli.StringFlag{
			Name:        "apns-private-key",
			Usage:       "APNs private key path",
			Sources:     cli.EnvVars("NOLET_APPLE_APNS_PRIVATE_KEY"),
			Destination: &LocalConfig.Apple.ApnsPrivateKey,
			Value: `-----BEGIN PRIVATE KEY-----
MIGTAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBHkwdwIBAQQgvjopbchDpzJNojnc
o7ErdZQFZM7Qxho6m61gqZuGVRigCgYIKoZIzj0DAQehRANCAAQ8ReU0fBNg+sA+
ZdDf3w+8FRQxFBKSD/Opt7n3tmtnmnl9Vrtw/nUXX4ldasxA2gErXR4YbEL9Z+uJ
REJP/5bp
-----END PRIVATE KEY-----`,
		},
		&cli.StringFlag{
			Name:        "topic",
			Usage:       "APNs topic",
			Sources:     cli.EnvVars("NOLET_APPLE_TOPIC"),
			Destination: &LocalConfig.Apple.Topic,
			Value:       "me.uuneo.Meoworld",
		},
		&cli.StringFlag{
			Name:        "key-id",
			Usage:       "APNs key ID",
			Sources:     cli.EnvVars("NOLET_APPLE_KEY_ID"),
			Destination: &LocalConfig.Apple.KeyID,
			Value:       "BNY5GUGV38",
		},
		&cli.StringFlag{
			Name:        "team-id",
			Usage:       "APNs team ID",
			Sources:     cli.EnvVars("NOLET_APPLE_TEAM_ID"),
			Destination: &LocalConfig.Apple.TeamID,
			Value:       "FUWV6U942Q",
		},
		&cli.BoolFlag{
			Name:        "develop",
			Usage:       "Use APNs development environment",
			Sources:     cli.EnvVars("NOLET_APPLE_DEVELOP"),
			Aliases:     []string{"dev"},
			Value:       false,
			Destination: &LocalConfig.Apple.Develop,
		},
		&cli.StringFlag{
			Name:        "ICP",
			Usage:       "Icp Footer Info",
			Sources:     cli.EnvVars("NOLET_ICP_INFO"),
			Aliases:     []string{"icp"},
			Destination: &LocalConfig.System.ICPInfo,
			Value:       "",
		},
		&cli.StringFlag{
			Name:    "config",
			Usage:   "Config file Dir",
			Aliases: []string{"c"},
			Value:   "",
		},
		&cli.BoolFlag{
			Name:        "proxy-download",
			Usage:       "Proxy Download",
			Sources:     cli.EnvVars("NOLET_PROXY_DOWNLOAD"),
			Aliases:     []string{"dp"},
			Value:       false,
			Destination: &LocalConfig.System.ProxyDownload,
		},
		&cli.StringFlag{
			Name:        "log-path",
			Usage:       "Log Path",
			Sources:     cli.EnvVars("NOLET_LOG_PATH"),
			Aliases:     []string{"lp"},
			Value:       "./data/logs/app.log",
			Destination: &LocalConfig.System.LogPath,
		},
		&cli.BoolFlag{
			Name:        "chttps",
			Usage:       "custom https",
			Sources:     cli.EnvVars("NOLET_CUSTOM_HTTPS"),
			Value:       false,
			Destination: &LocalConfig.System.CustomHttps,
		},

		&cli.BoolFlag{
			Name:    "out-config",
			Usage:   "export config",
			Sources: cli.EnvVars("NOLET_OUT_CONFIG"),
			Action: func(ctx context.Context, command *cli.Command, s bool) error {
				SynchronousFieldFile()
				return cli.Exit("create success ...", 0)
			},
		},
	}
}
