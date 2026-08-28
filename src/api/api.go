package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/hospitalapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/patientapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/staffapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/domain/userapp"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type BusConfig struct {
	UserBus            userbus.Business
	StaffBus           staffbus.Business
	PatientBus         patientbus.Business
	HospitalBus        hospitalbus.Business
	HospitalPatientBus hospitalpatientbus.Business
	HospitalStaffBus   hospitalstaffbus.Business
}

type Config struct {
	Log       *logger.Logger
	DB        *sqlx.DB
	Origins   []string
	BusConfig BusConfig
}

func Handler(r *gin.Engine, cfg Config) http.Handler {
	r.Use(
		mid.Cors(cfg.Origins),
		mid.Logger(cfg.Log),
	)

	userapp.Routes(r, userapp.Config{
		Log:     cfg.Log,
		DB:      cfg.DB,
		UserBus: cfg.BusConfig.UserBus,
	})

	staffapp.Routes(r, staffapp.Config{
		Log:              cfg.Log,
		DB:               cfg.DB,
		HospitalStaffBus: cfg.BusConfig.HospitalStaffBus,
		StaffBus:         cfg.BusConfig.StaffBus,
		UserBus:          cfg.BusConfig.UserBus,
	})

	patientapp.Routes(r, patientapp.Config{
		Log:        cfg.Log,
		DB:         cfg.DB,
		PatientBus: cfg.BusConfig.PatientBus,
	})

	hospitalapp.Routes(r, hospitalapp.Config{
		Log:         cfg.Log,
		DB:          cfg.DB,
		HospitalBus: cfg.BusConfig.HospitalBus,
	})

	return r
}
