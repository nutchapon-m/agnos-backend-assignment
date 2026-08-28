package staffapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log      *logger.Logger
	DB       *sqlx.DB
	StaffBus staffbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.StaffBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/staff", trans, api.Create)
	g.GET("/staff", api.Query)
	g.GET("/staff/:id", api.GetByID)
	g.DELETE("/staff/:id", trans, api.Delete)
}
