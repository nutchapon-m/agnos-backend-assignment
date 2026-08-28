package hospitalapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log         *logger.Logger
	DB          *sqlx.DB
	HospitalBus hospitalbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.HospitalBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/hospital", trans, api.Create)
	g.GET("/hospital", api.Query)
	g.GET("/hospital/:id", api.GetByID)
	g.DELETE("/hospital/:id", trans, api.Delete)
}
