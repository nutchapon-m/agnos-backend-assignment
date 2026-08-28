package hospitalpatientapp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

type App struct {
	hospitalPatientBus hospitalpatientbus.Business
}

func NewApp(hospitalPatientBus hospitalpatientbus.Business) *App {
	return &App{hospitalPatientBus: hospitalPatientBus}
}

func (a *App) Create(c *gin.Context) {
	var app NewHospitalPatient
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	newHospitalPatient, err := toNewHospitalPatient(app)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	bus, err := a.bus(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	hospitalPatient, err := bus.Create(c, newHospitalPatient)
	if err != nil {
		switch {
		case errors.Is(err, hospitalpatientbus.ErrDuplicate), errors.Is(err, sqldb.ErrDBDuplicatedEntry):
			response.Fail(c, http.StatusConflict, "Hospital Patient Already Exist", hospitalpatientbus.ErrDuplicate.Error())
		case errors.Is(err, hospitalpatientbus.ErrInvalidReference), errors.Is(err, sqldb.ErrDBForeignKeyViolation):
			response.Fail(c, http.StatusUnprocessableEntity, "Invalid Reference", hospitalpatientbus.ErrInvalidReference.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	response.OK(c, toAppHospitalPatient(hospitalPatient))
}

func (a *App) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	hospitalPatient, err := a.hospitalPatientBus.GetByID(c, id)
	if err != nil {
		if errors.Is(err, hospitalpatientbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Hospital Patient Not Found", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppHospitalPatient(hospitalPatient))
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

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, hospitalpatientbus.DefaultOrderBy)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	hospitalPatients, err := a.hospitalPatientBus.Query(c, parseFilter(qp), pg, orderBy)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	meta := response.Meta{
		Page:    pg.Number(),
		PerPage: pg.RowsPerPage(),
		Total:   len(hospitalPatients),
	}
	response.OKWithMeta(c, toAppHospitalPatients(hospitalPatients), meta)
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
		if errors.Is(err, hospitalpatientbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Hospital Patient Not Found", err.Error())
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
func (a *App) bus(c *gin.Context) (hospitalpatientbus.Business, error) {
	tx, ok := mid.GetTran(c)
	if !ok {
		return a.hospitalPatientBus, nil
	}

	return a.hospitalPatientBus.NewWithTx(tx)
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
