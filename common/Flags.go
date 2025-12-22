package common

import (
	"context"
	"log"
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
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.Addr = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "url-prefix",
			Usage:       "Serve URL Prefix",
			Sources:     cli.EnvVars("NOLET_SERVER_URL_PREFIX"),
			Value:       "/",
			Destination: &LocalConfig.System.URLPrefix,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.URLPrefix = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "dir",
			Usage:       "Server data storage dir",
			Sources:     cli.EnvVars("NOLET_SERVER_DATA_DIR"),
			Value:       "./data",
			Destination: &LocalConfig.System.DataDir,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.DataDir = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "dsn",
			Usage:       "MySQL DSN user:pass@tcp(host)/dbname",
			Sources:     cli.EnvVars("NOLET_SERVER_DSN"),
			Destination: &LocalConfig.System.DSN,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.DSN = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "cert",
			Usage:       "Server TLS certificate",
			Sources:     cli.EnvVars("NOLET_SERVER_CERT"),
			Destination: &LocalConfig.System.Cert,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.Cert = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "key",
			Usage:       "Server TLS certificate key",
			Sources:     cli.EnvVars("NOLET_SERVER_KEY"),
			Destination: &LocalConfig.System.Key,
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.Key = s
				return nil
			},
		},
		&cli.BoolFlag{
			Name:        "reduce-memory-usage",
			Usage:       "Aggressively reduces memory usage at the cost of higher CPU usage if set to true",
			Sources:     cli.EnvVars("NOLET_SERVER_REDUCE_MEMORY_USAGE"),
			Value:       false,
			Destination: &LocalConfig.System.ReduceMemoryUsage,
			Action: func(ctx context.Context, command *cli.Command, b bool) error {
				LocalConfig.System.ReduceMemoryUsage = b
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "user",
			Usage:       "Basic auth username",
			Sources:     cli.EnvVars("NOLET_SERVER_BASIC_AUTH_USER"),
			Aliases:     []string{"u"},
			Destination: &LocalConfig.System.User,
			Value:       "",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.User = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "password",
			Usage:       "Basic auth password",
			Sources:     cli.EnvVars("NOLET_SERVER_BASIC_AUTH_PASSWORD"),
			Aliases:     []string{"p"},
			Destination: &LocalConfig.System.Password,
			Value:       "",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.Password = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "push-password",
			Usage:       "push auth password",
			Sources:     cli.EnvVars("NOLET_PUSH_PASSWORD"),
			Destination: &LocalConfig.System.PushPassword,
			Value:       "",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.PushPassword = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "sign-key",
			Usage:       "App Sign Key",
			Sources:     cli.EnvVars("NOLET_SIGN_KEY"),
			Aliases:     []string{"sk"},
			Destination: &LocalConfig.System.SignKey,
			Value:       "",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.SignKey = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "proxy-header",
			Usage:       "The remote IP address used by the NOLET server http header",
			Sources:     cli.EnvVars("NOLET_SERVER_PROXY_HEADER"),
			Destination: &LocalConfig.System.ProxyHeader,
			Value:       "",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.System.ProxyHeader = s
				return nil
			},
		},
		&cli.IntFlag{
			Name:        "max-batch-push-count",
			Usage:       "Maximum number of batch pushes allowed, -1 means no limit",
			Sources:     cli.EnvVars("NOLET_SERVER_MAX_BATCH_PUSH_COUNT"),
			Value:       -1,
			Destination: &LocalConfig.System.MaxBatchPushCount,
			Action: func(ctx context.Context, command *cli.Command, v int) error {
				LocalConfig.System.MaxBatchPushCount = v
				return nil
			},
		},
		&cli.IntFlag{
			Name:        "max-apns-client-count",
			Usage:       "Maximum number of APNs client connections",
			Sources:     cli.EnvVars("NOLET_SERVER_MAX_APNS_CLIENT_COUNT"),
			Aliases:     []string{"max"},
			Value:       1,
			Destination: &LocalConfig.System.MaxAPNSClientCount,
			Action: func(ctx context.Context, command *cli.Command, v int) error {
				LocalConfig.System.MaxAPNSClientCount = v
				return nil
			},
		},
		&cli.IntFlag{
			Name:        "max-device-key-arr-length",
			Usage:       "Maximum number of deviceKey list length connections",
			Sources:     cli.EnvVars("NOLET_CONCURRENCY"),
			Value:       10,
			Destination: &LocalConfig.System.MaxDeviceKeyArrLength,
			Action: func(ctx context.Context, command *cli.Command, b int) error {
				LocalConfig.System.MaxDeviceKeyArrLength = b
				return nil
			},
		},
		&cli.IntFlag{
			Name:        "concurrency",
			Usage:       "Maximum number of concurrent connections",
			Sources:     cli.EnvVars("NOLET_SERVER_CONCURRENCY"),
			Value:       256 * 1024,
			Hidden:      true,
			Destination: &LocalConfig.System.Concurrency,
			Action: func(ctx context.Context, command *cli.Command, b int) error {
				LocalConfig.System.Concurrency = b
				return nil
			},
		},
		&cli.DurationFlag{
			Name:        "read-timeout",
			Usage:       "The amount of time allowed to read the full request, including the body",
			Sources:     cli.EnvVars("NOLET_SERVER_READ_TIMEOUT"),
			Value:       3 * time.Second,
			Hidden:      true,
			Destination: &LocalConfig.System.ReadTimeout,
			Action: func(ctx context.Context, command *cli.Command, duration time.Duration) error {
				LocalConfig.System.ReadTimeout = duration
				return nil
			},
		},
		&cli.DurationFlag{
			Name:        "write-timeout",
			Usage:       "The maximum duration before timing out writes of the response",
			Sources:     cli.EnvVars("NOLET_SERVER_WRITE_TIMEOUT"),
			Value:       10 * time.Second,
			Hidden:      true,
			Destination: &LocalConfig.System.WriteTimeout,
			Action: func(ctx context.Context, command *cli.Command, duration time.Duration) error {
				LocalConfig.System.WriteTimeout = duration
				return nil
			},
		},
		&cli.DurationFlag{
			Name:        "idle-timeout",
			Usage:       "The maximum amount of time to wait for the next request when keep-alive is enabled",
			Sources:     cli.EnvVars("NOLET_SERVER_IDLE_TIMEOUT"),
			Value:       10 * time.Second,
			Hidden:      true,
			Destination: &LocalConfig.System.IdleTimeout,
			Action: func(ctx context.Context, command *cli.Command, duration time.Duration) error {
				LocalConfig.System.IdleTimeout = duration
				return nil
			},
		},
		&cli.BoolFlag{
			Name:        "debug",
			Value:       false,
			Usage:       "enable debug mode",
			Sources:     cli.EnvVars("NOLET_DEBUG"),
			Destination: &LocalConfig.System.Debug,
			Action: func(ctx context.Context, command *cli.Command, b bool) error {
				LocalConfig.System.Debug = b
				return nil
			},
		},
		&cli.StringSliceFlag{
			Name:        "auths",
			Value:       []string{},
			Usage:       "auth id list",
			Sources:     cli.EnvVars("NOLET_AUTHS"),
			Destination: &LocalConfig.System.Auths,
			Action: func(ctx context.Context, command *cli.Command, strings []string) error {
				LocalConfig.System.Auths = strings
				return nil
			},
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
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.Apple.ApnsPrivateKey = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "topic",
			Usage:       "APNs topic",
			Sources:     cli.EnvVars("NOLET_APPLE_TOPIC"),
			Destination: &LocalConfig.Apple.Topic,
			Value:       "me.uuneo.Meoworld",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.Apple.Topic = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "key-id",
			Usage:       "APNs key ID",
			Sources:     cli.EnvVars("NOLET_APPLE_KEY_ID"),
			Destination: &LocalConfig.Apple.KeyID,
			Value:       "BNY5GUGV38",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.Apple.KeyID = s
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "team-id",
			Usage:       "APNs team ID",
			Sources:     cli.EnvVars("NOLET_APPLE_TEAM_ID"),
			Destination: &LocalConfig.Apple.TeamID,
			Value:       "FUWV6U942Q",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				LocalConfig.Apple.TeamID = s
				return nil
			},
		},
		&cli.BoolFlag{
			Name:        "develop",
			Usage:       "Use APNs development environment",
			Sources:     cli.EnvVars("NOLET_APPLE_DEVELOP"),
			Aliases:     []string{"dev"},
			Value:       false,
			Destination: &LocalConfig.Apple.Develop,
			Action: func(ctx context.Context, command *cli.Command, b bool) error {
				LocalConfig.Apple.Develop = b
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "ICP",
			Usage:       "Icp Footer Info",
			Sources:     cli.EnvVars("NOLET_ICP_INFO"),
			Aliases:     []string{"icp"},
			Destination: &LocalConfig.System.ICPInfo,
			Value:       "",
			Action: func(ctx context.Context, command *cli.Command, s string) error {
				log.Println(s)
				LocalConfig.System.ICPInfo = s
				return nil
			},
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
			Action: func(ctx context.Context, command *cli.Command, b bool) error {
				LocalConfig.System.ProxyDownload = b
				return nil
			},
		},
		&cli.StringFlag{
			Name:        "export-path",
			Usage:       "Export Database",
			Sources:     cli.EnvVars("NOLET_EXPORT_PATH"),
			Aliases:     []string{"dc"},
			Value:       "",
			Destination: &LocalConfig.System.ExportPath,
		},
		&cli.StringFlag{
			Name:        "import-path",
			Usage:       "Export Database",
			Sources:     cli.EnvVars("NOLET_IMPORT_PATH"),
			Aliases:     []string{"dr"},
			Value:       "",
			Destination: &LocalConfig.System.ExportPath,
		},

		&cli.BoolFlag{
			Name:        "chttps",
			Usage:       "custom https",
			Sources:     cli.EnvVars("NOLET_CUSTOM_HTTPS"),
			Value:       false,
			Destination: &LocalConfig.System.CustomHttps,
			Action: func(ctx context.Context, command *cli.Command, b bool) error {
				LocalConfig.System.CustomHttps = b
				return nil
			},
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
