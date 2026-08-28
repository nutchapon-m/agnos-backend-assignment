package userapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log     *logger.Logger
	DB      *sqlx.DB
	UserBus userbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.UserBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/user", trans, api.Create)
	g.GET("/user", api.Query)
	g.GET("/user/:id", api.GetByID)
	g.DELETE("/user/:id", trans, api.Delete)
}
