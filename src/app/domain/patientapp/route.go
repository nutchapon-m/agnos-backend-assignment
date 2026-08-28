package patientapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log        *logger.Logger
	DB         *sqlx.DB
	PatientBus patientbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.PatientBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/patient", trans, api.Create)
	g.GET("/patient", api.Query)
	g.GET("/patient/:id", api.GetByID)
	g.DELETE("/patient/:id", trans, api.Delete)
}
