package patientapp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalpatientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/patientbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

type App struct {
	patientBus         patientbus.Business
	hospitalPatientBus hospitalpatientbus.Business
}

func NewApp(patientBus patientbus.Business, hospitalPatientBus hospitalpatientbus.Business) *App {
	return &App{patientBus: patientBus, hospitalPatientBus: hospitalPatientBus}
}

func (a *App) Create(c *gin.Context) {
	var app NewPatient
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	newPatient, err := toNewPatient(app)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	bus, err := a.bus(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	patient, err := bus.Create(c, newPatient)
	if err != nil {
		if errors.Is(err, patientbus.ErrDuplicate) || errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			response.Fail(c, http.StatusConflict, "Patient Already Exist", patientbus.ErrDuplicate.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppPatient(patient))
}

func (a *App) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	patient, err := a.patientBus.GetByID(c, id)
	if err != nil {
		if errors.Is(err, patientbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Patient Not Found", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	hp, err := a.hospitalPatientBus.Query(c, hospitalpatientbus.QueryFilter{PatientID: &id}, page.Page{}, hospitalpatientbus.DefaultOrderBy)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	patientApp := toAppPatient(patient)
	if len(hp) > 0 {
		patientApp.HospitalNumber = hp[len(hp)-1].HN
	}

	response.OK(c, patientApp)
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

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, patientbus.DefaultOrderBy)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	patients, err := a.patientBus.Query(c, parseFilter(qp), pg, orderBy)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	meta := response.Meta{
		Page:    pg.Number(),
		PerPage: pg.RowsPerPage(),
		Total:   len(patients),
	}
	response.OKWithMeta(c, toAppPatients(patients), meta)
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
		if errors.Is(err, patientbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Patient Not Found", err.Error())
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
func (a *App) bus(c *gin.Context) (patientbus.Business, error) {
	tx, ok := mid.GetTran(c)
	if !ok {
		return a.patientBus, nil
	}

	return a.patientBus.NewWithTx(tx)
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
