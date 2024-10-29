package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/BasedDevelopment/eve/pkg/fwdlog"
	"github.com/ezrizhu/bgprestlg/internal/config"
	"github.com/ezrizhu/bgprestlg/internal/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var debug bool
var jsonLog bool
var restport int
var bgpport int
var logFormat string
var logLevel string

var configPath string

const (
	version = "0.0.1"
)

func init() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	flag.IntVar(&bgpport, "bgpport", 179, "bgpport")
	flag.IntVar(&restport, "restport", 8080, "restport")
	flag.BoolVar(&debug, "debug", false, "debug")
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

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug log enabled")
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if err := config.Load(configPath); err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().
		Int("bgp port", bgpport).
		Int("rest port", restport).
		Msg("Starting BGP REST LG")

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:     ":" + strconv.Itoa(restport),
		Handler:  server.Handler(),
		ErrorLog: fwdlog.Logger(),
	}

	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("Failed to start HTTP server")
		}
	}()
	log.Info().
		Msg("Started BGP server and Webserver")

	<-done
	log.Info().Msg("Stopping")
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	//if err := bgpStop(shutdownCtx); err != nil {
	//	log.Fatal().Err(err).Msg("Failed to gracefully stop bgp instance")
	//}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal().Err(err).Msg("Failed to gracefully stop http server")
	}
	log.Info().Msg("Graceful Shutdown Successful, bye")
}
