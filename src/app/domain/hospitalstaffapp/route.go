package hospitalstaffapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log              *logger.Logger
	DB               *sqlx.DB
	HospitalStaffBus hospitalstaffbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.HospitalStaffBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/hospital-staff", trans, api.Create)
	g.GET("/hospital-staff", api.Query)
	g.GET("/hospital-staff/:id", api.GetByID)
	g.DELETE("/hospital-staff/:id", trans, api.Delete)
}
