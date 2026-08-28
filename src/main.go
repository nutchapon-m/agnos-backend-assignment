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
	"github.com/nutchapon-m/agnos-backend-assignment/src/api"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus/stores/hospitaldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus/stores/hospitalpatientdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus/stores/hospitalstaffdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus/stores/patientdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus/stores/staffdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus/stores/userdb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/migrate"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/env"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
	"github.com/spf13/viper"
)

var (
	migration = flag.Bool("migration", false, "Run migrations file on start app")
)

const (
	addr = ":8000"
)

func main() {
	// -------------------------------------------------------------------------
	// Flag on start service

	flag.Parse()

	// -------------------------------------------------------------------------
	// Load ENV

	env.Load(".", "config.yml")

	log := logger.New(os.Stdout, logger.LevelInfo, "agnos test")

	ctx := context.Background()
	if err := run(ctx, log); err != nil {
		log.Error(ctx, "error startup", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, log *logger.Logger) error {
	// -------------------------------------------------------------------------
	// GOMAXPROCS

	log.Info(ctx, "startup", "GOMAXPROCS", runtime.GOMAXPROCS(0))

	// -------------------------------------------------------------------------
	// Database support

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

	userBus := userbus.NewBusiness(log, userdb.NewStore(db))
	staffBus := staffbus.NewBusiness(log, staffdb.NewStore(db))
	patientBus := patientbus.NewBusiness(log, patientdb.NewStore(db))
	hospitalBus := hospitalbus.NewBusiness(log, hospitaldb.NewStore(db))
	hospitalPatientBus := hospitalpatientbus.NewBusiness(log, hospitalpatientdb.NewStore(db))
	hospitalStaffBus := hospitalstaffbus.NewBusiness(log, hospitalstaffdb.NewStore(db))

	conf := api.Config{
		Log:     log,
		DB:      db,
		Origins: viper.GetStringSlice("cors_origins"),
		BusConfig: api.BusConfig{
			UserBus:            userBus,
			StaffBus:           staffBus,
			PatientBus:         patientBus,
			HospitalBus:        hospitalBus,
			HospitalPatientBus: hospitalPatientBus,
			HospitalStaffBus:   hospitalStaffBus,
		},
	}

	r := gin.Default()

	server := http.Server{
		Addr:         addr,
		Handler:      api.Handler(r, conf),
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

	// -------------------------------------------------------------------------
	// Shutdown

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
