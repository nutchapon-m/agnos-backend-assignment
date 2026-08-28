package hospitalapp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

type App struct {
	hospitalBus hospitalbus.Business
}

func NewApp(hospitalBus hospitalbus.Business) *App {
	return &App{hospitalBus: hospitalBus}
}

func (a *App) Create(c *gin.Context) {
	var app NewHospital
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	bus, err := a.bus(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	hospital, err := bus.Create(c, toNewHospital(app))
	if err != nil {
		if errors.Is(err, hospitalbus.ErrDuplicate) || errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			response.Fail(c, http.StatusConflict, "Hospital Already Exist", hospitalbus.ErrDuplicate.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppHospital(hospital))
}

func (a *App) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	hospital, err := a.hospitalBus.GetByID(c, id)
	if err != nil {
		if errors.Is(err, hospitalbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Hospital Not Found", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppHospital(hospital))
}

func (a *App) Query(c *gin.Context) {
	var qp queryParams
	if err := c.ShouldBindQuery(&qp); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	pg, err := page.Parse(qp.Page, qp.Limit)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, hospitalbus.DefaultOrderBy)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	hospitals, err := a.hospitalBus.Query(c, parseFilter(qp), pg, orderBy)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	meta := response.Meta{
		Page:    pg.Number(),
		PerPage: pg.RowsPerPage(),
		Total:   len(hospitals),
	}
	response.OKWithMeta(c, toAppHospitals(hospitals), meta)
}

func (a *App) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	bus, err := a.bus(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	if _, err := bus.GetByID(c, id); err != nil {
		if errors.Is(err, hospitalbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Hospital Not Found", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	if err := bus.Delete(c, id); err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, nil)
}

// bus returns a business value scoped to the request transaction when the
// BeginCommitRollback middleware is in play, otherwise the default one.
func (a *App) bus(c *gin.Context) (hospitalbus.Business, error) {
	tx, ok := mid.GetTran(c)
	if !ok {
		return a.hospitalBus, nil
	}

	return a.hospitalBus.NewWithTx(tx)
}

func parseID(c *gin.Context) (int, error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return 0, errors.New("id must be a number")
	}

	if id <= 0 {
		return 0, errors.New("id must be larger than 0")
	}

	return id, nil
}
