package main

import (
	"context"
	"crypto/rsa"
	"expvar"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/danielmbirochi/go-sample-service/app/services/sales-api/handlers"
	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/danielmbirochi/go-sample-service/foundation/database"
	"github.com/danielmbirochi/go-sample-service/foundation/logger"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/automaxprocs/maxprocs"
	"go.uber.org/zap"
)

var build = "develop"

func main() {
	// log := log.New(os.Stdout, "SALES: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)
	log, err := logger.New("SALES-API")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer log.Sync()

	if err := run(log); err != nil {
		log.Errorw("main: error: ", err)
		os.Exit(1)
	}
}

func run(log *zap.SugaredLogger) error {

	// =========================================================================
	// GOMAXPROCS

	// Set the correct number of threads for the service
	// based on what is available either by the machine or cluster quotas.
	if _, err := maxprocs.Set(); err != nil {
		return fmt.Errorf("maxprocs: %w", err)
	}
	log.Infow("startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// ============================================================================================
	// Setup Configutarion
	var cfg struct {
		conf.Version
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:3000"` // noprint - this tag is used for hidding the config prop from stdout
			DebugHost       string        `conf:"default:0.0.0.0:4000"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s"`
		}
		Auth struct {
			KeyID          string `conf:"default:32bc1165-24t2-61a7-af3e-9da4agf2h1p1"`
			PrivateKeyFile string `conf:"default:/app/private.pem"`
			Algorithm      string `conf:"default:RS256"`
		}
		DB struct {
			User       string `conf:"default:testuser"`
			Password   string `conf:"default:mysecretpassword,mask"`
			Hostname   string `conf:"default:0.0.0.0"`
			Name       string `conf:"default:testdb"`
			DisableTLS bool   `conf:"default:false"`
		}
		Zipkin struct {
			ReporterURI string  `conf:"default:http://localhost:9411/api/v2/spans"`
			ServiceName string  `conf:"default:sales-api"`
			Probability float64 `conf:"default:0.05"`
		}
	}

	cfg.Version.SVN = build
	cfg.Version.Desc = "Go Sample service"

	if err := conf.Parse(os.Args[1:], "SALES", &cfg); err != nil {
		switch err {

		case conf.ErrHelpWanted:
			usage, err := conf.Usage("SALES", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}

			fmt.Println(usage)
			return nil

		case conf.ErrVersionWanted:
			version, err := conf.VersionString("SALES", &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}

			fmt.Println(version)
			return nil

		}

		return errors.Wrap(err, "parsing config")
	}

	// ============================================================================================
	// App Starting

	expvar.NewString("build").Set(build)
	log.Infow("starting service", "version", build)
	defer log.Infow("shutdown complete")

	configOut, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Infow("startup", "config", configOut)

	// ============================================================================================
	// Initialize authentication support
	log.Infow("startup", "status", "initializing authentication support")

	privatePEM, err := ioutil.ReadFile(cfg.Auth.PrivateKeyFile)
	if err != nil {
		return errors.Wrap(err, "reading auth pem file")
	}

	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privatePEM)
	if err != nil {
		return errors.Wrap(err, "parsing auth pem file")
	}

	// In production the Key pair set (Auth.PrivateKeyFile) would be get from external services.
	// For local development env we`re using the local file system for that.
	lookupKey := func(kid string) (*rsa.PublicKey, error) {
		switch kid {
		case cfg.Auth.KeyID:
			return &privateKey.PublicKey, nil
		}
		return nil, errors.Errorf("no key pair found for the specified kid: %s", kid)
	}

	auth, err := auth.New(cfg.Auth.Algorithm, lookupKey, auth.Keys{cfg.Auth.KeyID: privateKey})
	if err != nil {
		return errors.Wrap(err, "initializing auth service")
	}

	// =========================================================================
	// Start Database

	log.Infow("startup", "status", "initializing database support", "host", cfg.DB.Hostname)

	cfg.DB.DisableTLS = true

	db, err := database.Open(database.Config{
		User:       cfg.DB.User,
		Password:   cfg.DB.Password,
		Hostname:   cfg.DB.Hostname,
		Name:       cfg.DB.Name,
		DisableTLS: cfg.DB.DisableTLS,
	})
	if err != nil {
		return errors.Wrap(err, "connecting to db")
	}
	defer func() {
		log.Infow("shutdown", "status", "stopping database support", "host", cfg.DB.Hostname)
		db.Close()
	}()

	// =========================================================================
	// Start Tracing Support

	// WARNING: The current Init settings are using defaults which may not be
	// compatible with your project. Please review the documentation for
	// opentelemetry.

	log.Infow("startup", "status", "initializing OT/Zipkin tracing support")

	exporter, err := zipkin.New(
		cfg.Zipkin.ReporterURI,
		zipkin.WithLogger(zap.NewStdLog(log.Desugar())),
	)
	if err != nil {
		return errors.Wrap(err, "creating new zipkin exporter")
	}

	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.TraceIDRatioBased(cfg.Zipkin.Probability)),
		trace.WithBatcher(exporter,
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
			trace.WithBatchTimeout(trace.DefaultBatchTimeout),
			trace.WithMaxExportBatchSize(trace.DefaultMaxExportBatchSize),
		),
		trace.WithResource(
			resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(cfg.Zipkin.ServiceName),
				attribute.String("exporter", "zipkin"),
			),
		),
	)

	otel.SetTracerProvider(tp)
	defer tp.Shutdown(context.Background())

	// ============================================================================================
	// Start Debug Service
	log.Infow("startup", "status", "debug router started", "host", cfg.Web.DebugHost)

	// The DebugStandardLibraryMux func returns a mux to listen and serve on for all
	// the debug related endpoints. It includes the standard library endpoints, such as:
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Construct the mux for debug calls.
	debugMux := handlers.DebugStandardLibraryMux()

	// Not concerned with shutting this down when the application is shutdown.
	go func() {
		if err := http.ListenAndServe(cfg.Web.DebugHost, debugMux); err != nil {
			log.Errorw("shutdown", "status", "debug router closed", "host", cfg.Web.DebugHost, "ERROR", err)
		}
	}()

	// ============================================================================================
	// Start API Service
	log.Infow("startup", "status", "initializing API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handlers.API(build, shutdown, log, auth, db),
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		ErrorLog:     zap.NewStdLog(log.Desugar()),
	}

	// Make a channel for listening errors coming from the API Http listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	go func() {
		log.Infow("startup", "status", "api router started", "host", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Infow("shutdown", "status", "shutdown started", "signal", sig)
		defer log.Infow("shutdown", "status", "shutdown complete", "signal", sig)

		// Give outstanding requests a deadline for completion.
		ctx, cancel := context.WithTimeout(context.Background(), cfg.Web.ShutdownTimeout)
		defer cancel()

		// Asking listener to shutdown and shed load.
		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return errors.Wrap(err, "could not stop server gracefully!")
		}
	}

	return nil
}
