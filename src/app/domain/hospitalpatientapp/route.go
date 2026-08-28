package hospitalpatientapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log                *logger.Logger
	DB                 *sqlx.DB
	HospitalPatientBus hospitalpatientbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.HospitalPatientBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/hospital-patient", trans, api.Create)
	g.GET("/hospital-patient", api.Query)
	g.GET("/hospital-patient/:id", api.GetByID)
	g.DELETE("/hospital-patient/:id", trans, api.Delete)
}
