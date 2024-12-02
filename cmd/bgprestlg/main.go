package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/BasedDevelopment/eve/pkg/fwdlog"
	"github.com/ezrizhu/bgprestlg/internal/bgp"
	"github.com/ezrizhu/bgprestlg/internal/config"
	"github.com/ezrizhu/bgprestlg/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var debugLog bool
var jsonLog bool
var logFormat string
var logLevel string

var configPath string

const (
	version = "0.0.1"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	flag.BoolVar(&debugLog, "debug", false, "debug")
	flag.BoolVar(&jsonLog, "json", false, "json logging")
	flag.StringVar(&configPath, "config", "config.toml", "config file")
	flag.StringVar(&logLevel, "log-level", "debug", "Log level (trace, debug, info, warn, error, fatal, panic)")
	flag.StringVar(&logFormat, "log-format", "json", "Log format (json, pretty)")
}

func main() {
	flag.Parse()
	configureLogger()

	if !jsonLog {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	if debugLog {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug log enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if err := config.Load(configPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().
		Str("rest address", config.Config.API.Address).
		Int("rest port", config.Config.API.Port).
		Str("bgp address", config.Config.BGP.Address).
		Int("bgp port", config.Config.BGP.Port).
		Msg("Starting BGP REST LG")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:     config.Config.API.Address + ":" + strconv.Itoa(config.Config.API.Port),
		Handler:  server.Handler(),
		ErrorLog: fwdlog.Logger(),
	}

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()
	bgp.SrvInit()
	log.Info().
		Msg("Started BGP server and Webserver")

	go startMemoryCleanup()

	<-done
	log.Info().Msg("Stopping")
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := bgp.SrvStop(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Failed to gracefully stop bgp instance")
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Failed to gracefully stop http server")
	}
	log.Info().Msg("Graceful Shutdown Successful, bye")
}

func startMemoryCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		debug.FreeOSMemory()
	}
}
