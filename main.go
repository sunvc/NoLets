package main

import (
	"context"
	"crypto/tls"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sunvc/NoLets/common"
	"github.com/sunvc/NoLets/controller"
	"github.com/sunvc/NoLets/database"
	"github.com/sunvc/NoLets/push"
	"github.com/sunvc/NoLets/router"
	"github.com/urfave/cli/v3"
)

var (
	version   string
	buildDate string
	commitID  string
)

//go:embed static/*
var staticFS embed.FS

func main() {
	// Create context that listens for the interrupt signal from the OS.
	ctxOut, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	defer stop()

	common.StaticFS = &staticFS
	app := &cli.Command{
		Name:    "NoLets",
		Usage:   "Push Server For NoLet",
		Flags:   common.Flags(),
		Authors: []any{"to@uuneo.com"},
		Action: func(_ context.Context, command *cli.Command) error {
			if common.LocalConfig.System.CustomHttps {
				controller.CreateSSL()
				common.LocalConfig.System.Addr = "0.0.0.0:8443"
			}

			if _, err := os.Stat(common.BaseDir()); os.IsNotExist(err) {
				if err = os.MkdirAll(common.BaseDir(), 0755); err != nil {
					log.Println(fmt.Sprintf("failed to create database storage dir(%s): %v", common.BaseDir(), err))
					panic("failed to create database storage dir")
				}
			}

			if configPath := command.String("config"); configPath != "" {
				common.LocalConfig.SetConfig(configPath)
			}

			common.SetDefaultVersionOrCommID(version, buildDate, commitID)
			database.InitDatabase()

			systemConfig := common.LocalConfig.System

			if systemConfig.Debug {
				gin.SetMode(gin.DebugMode)
				gin.ForceConsoleColor()
			} else {
				gin.SetMode(gin.ReleaseMode)
			}

			engine := gin.New()

			tmpl := template.Must(template.New("").ParseFS(staticFS, "static/*.html"))
			engine.SetHTMLTemplate(tmpl)

			push.CreateAPNSClient(systemConfig.MaxAPNSClientCount)

			router.SetupRouter(engine)

			var tLSConfig *tls.Config

			if systemConfig.Key != "" && systemConfig.Cert != "" {

				cert, err := tls.LoadX509KeyPair(systemConfig.Cert, systemConfig.Key)

				if err != nil {
					log.Printf("failed to load TLS cert (cert=%s, key=%s): %v", systemConfig.Cert, systemConfig.Key, err)
				} else {
					tLSConfig = &tls.Config{
						Certificates: []tls.Certificate{cert},
						MinVersion:   tls.VersionTLS12,
					}
				}
			}

			server := &http.Server{
				Addr:           systemConfig.Addr,
				Handler:        engine,
				TLSConfig:      tLSConfig,
				ReadTimeout:    systemConfig.ReadTimeout,
				WriteTimeout:   systemConfig.WriteTimeout,
				IdleTimeout:    systemConfig.IdleTimeout,
				MaxHeaderBytes: 1 << 12,
			}
			httpServerError := make(chan error, 1)
			go func() {
				if tLSConfig != nil {
					if err := server.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
						httpServerError <- err
					}
				} else {
					if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
						httpServerError <- err
					}
				}
			}()

			select {
			case <-ctxOut.Done():
				log.Println("Received shutdown signal")
			case e := <-httpServerError:
				log.Printf("Server start error: %v", e)
			}

			ctxShutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			// Close HTTP server
			if err := server.Shutdown(ctxShutdown); err != nil {
				log.Printf("Server forced to shutdown error: %v", err)
			}

			// Close database connection
			if err := database.DB.Close(); err != nil {
				log.Printf("Database close error: %v", err)
			}

			// Close APNS client resources
			push.CloseAPNSClients()

			log.Println("All resources have been properly released")
			return nil
		},
	}

	_ = app.Run(context.Background(), os.Args)
}
