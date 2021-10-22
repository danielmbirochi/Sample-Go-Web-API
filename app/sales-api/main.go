package main

import (
	"context"
	"crypto/rsa"
	"expvar"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/danielmbirochi/go-sample-service/app/sales-api/handlers"
	"github.com/danielmbirochi/go-sample-service/business/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/pkg/errors"
)

var build = "develop"

func main() {
	log := log.New(os.Stdout, "SALES: ", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	if err := run(log); err != nil {
		log.Println("main: error: ", err)
		os.Exit(1)
	}
}

func run(log *log.Logger) error {

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
			KeyID          string `conf:"default:54bb2165-71e1-41a6-af3e-7da4a0e1e2c1"`
			PrivateKeyFile string `conf:"default:/Users/miranda/Documents/estudos/go/go-sample-service/private.pem"` // This is my local dev env, after moves to K8s it needs to reflects the correct path
			Algorithm      string `conf:"default:RS256"`
		}
	}

	cfg.Version.SVN = build
	cfg.Version.Desc = "Sample Go service"

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
	log.Printf("main: Started: Application initializing: version %q", build)
	defer log.Println("main: Completed")

	configOut, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main: Config: \n%v\n", configOut)

	// ============================================================================================
	// Initialize authentication support
	log.Println("main : Started : Initializing authentication support")

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

	// ============================================================================================
	// Start Debug Service
	//
	// /debug/pprof - Added to the default mux by importing the net/http/pprof package.
	// /debug/vars - Added to the default mux by importing the expvar package.
	//
	// Not concerned with shutting this down when the application is shutdown.
	log.Println("main: Initializing debugging support")
	go func() {
		log.Printf("main: Debug Listening %s", cfg.Web.DebugHost)
		if err := http.ListenAndServe(cfg.Web.DebugHost, http.DefaultServeMux); err != nil {
			log.Printf("main: Debug Listener closed : %v", err)
		}
	}()

	// ============================================================================================
	// Start API Service
	log.Println("main: Initializing API support")

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handlers.API(build, shutdown, log, auth),
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
	}

	// Make a channel for listening errors coming from the API Http listener. Use a
	// buffered channel so the goroutine can exit if we don't collect this error.
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("main: API listening on %s", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	// =========================================================================
	// Shutdown

	// Blocking main and waiting for shutdown.
	select {
	case err := <-serverErrors:
		return errors.Wrap(err, "server error")

	case sig := <-shutdown:
		log.Printf("main: %v : Starting shutdown", sig)

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
