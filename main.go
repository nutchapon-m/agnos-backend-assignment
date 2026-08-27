package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/sdk/env"
	"github.com/nutchapon-m/agnos-backend-assignment/sdk/logger"
	"github.com/nutchapon-m/agnos-backend-assignment/sdk/migrate"
	"github.com/nutchapon-m/agnos-backend-assignment/sdk/mux"
	"github.com/nutchapon-m/agnos-backend-assignment/sdk/sqldb"
	"github.com/spf13/viper"
)

var (
	migration = flag.Bool("migration", false, "Run migrations file on start app")
)

const (
	addr = ":8000"
)

func main() {
	flag.Parse()

	env.Load(".", "config.yaml")

	log := logger.New(os.Stdout, logger.LevelInfo, "test-api")
	ctx := context.Background()

	if err := run(ctx, log); err != nil {
		log.Error(ctx, "error", "detail", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	db, err := sqldb.Open(sqldb.Config{
		User:         viper.GetString("db.user"),
		Password:     viper.GetString("db.password"),
		Host:         viper.GetString("db.host"),
		Name:         viper.GetString("db.name"),
		Schema:       viper.GetString("db.schema"),
		MaxIdleConns: viper.GetInt("db.max_idle_conns"),
		MaxOpenConns: viper.GetInt("db.max_open_conns"),
		DisableTLS:   viper.GetBool("db.disable_tls"),
	})
	if err != nil {
		log.Error(ctx, "Error connect database")
		return err
	}
	defer db.Close()

	if err := sqldb.StatusCheck(ctx, db); err != nil {
		log.Error(ctx, "Database status check failed")
		return err
	}

	if *migration {
		log.Info(ctx, "Running migrations file...")
		if err := migrate.Run(db.DB, "up"); err != nil {
			log.Error(ctx, "error run migration file")
			return err
		}
		log.Info(ctx, "Migrate complete.")
	}

	r := gin.Default()

	mux.NewMux(r, mux.Config{
		DB:  db,
		Log: log,
	})

	server := http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverErrors := make(chan error, 1)
	go func() {
		log.Info(ctx, "service running", "addr", addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			server.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}
