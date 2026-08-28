package staffapp

import (
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
	"github.com/nutchapon-m/agnos-backend-assignment/src/foundation/logger"
)

type Config struct {
	Log              *logger.Logger
	DB               *sqlx.DB
	HospitalStaffBus hospitalstaffbus.Business
	StaffBus         staffbus.Business
	UserBus          userbus.Business
}

func Routes(r *gin.Engine, cfg Config) {
	api := NewApp(cfg.HospitalStaffBus, cfg.StaffBus, cfg.UserBus)

	trans := mid.BeginCommitRollback(cfg.Log, sqldb.NewBeginner(cfg.DB))

	g := r.Group("/api/v1")

	g.POST("/staff/create", trans, api.Create)
	g.POST("/staff/login", api.Login)
}
