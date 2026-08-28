package patientapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log                *logger.Logger
	DB                 *sqlx.DB
	PatientBus         patientbus.Business
	HospitalPatientBus hospitalpatientbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.PatientBus, cfg.HospitalPatientBus)

	g := r.Group("/api/v1")

	g.GET("/patient/:id", api.GetByID)
	g.GET("/patient/search", api.Query)
}
