package hospitalstaffapp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

type App struct {
	hospitalStaffBus hospitalstaffbus.Business
}

func NewApp(hospitalStaffBus hospitalstaffbus.Business) *App {
	return &App{hospitalStaffBus: hospitalStaffBus}
}

func (a *App) Create(c *gin.Context) {
	var app NewHospitalStaff
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	newHospitalStaff, err := toNewHospitalStaff(app)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	bus, err := a.bus(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	hospitalStaff, err := bus.Create(c, newHospitalStaff)
	if err != nil {
		switch {
		case errors.Is(err, hospitalstaffbus.ErrInvalidRole):
			response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		case errors.Is(err, hospitalstaffbus.ErrDuplicate), errors.Is(err, sqldb.ErrDBDuplicatedEntry):
			response.Fail(c, http.StatusConflict, "Hospital Staff Already Exist", hospitalstaffbus.ErrDuplicate.Error())
		case errors.Is(err, hospitalstaffbus.ErrInvalidReference), errors.Is(err, sqldb.ErrDBForeignKeyViolation):
			response.Fail(c, http.StatusUnprocessableEntity, "Invalid Reference", hospitalstaffbus.ErrInvalidReference.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	response.OK(c, toAppHospitalStaff(hospitalStaff))
}

func (a *App) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	hospitalStaff, err := a.hospitalStaffBus.GetByID(c, id)
	if err != nil {
		if errors.Is(err, hospitalstaffbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Hospital Staff Not Found", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppHospitalStaff(hospitalStaff))
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

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, hospitalstaffbus.DefaultOrderBy)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	hospitalStaffs, err := a.hospitalStaffBus.Query(c, parseFilter(qp), pg, orderBy)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	meta := response.Meta{
		Page:    pg.Number(),
		PerPage: pg.RowsPerPage(),
		Total:   len(hospitalStaffs),
	}
	response.OKWithMeta(c, toAppHospitalStaffs(hospitalStaffs), meta)
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
		if errors.Is(err, hospitalstaffbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Hospital Staff Not Found", err.Error())
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
func (a *App) bus(c *gin.Context) (hospitalstaffbus.Business, error) {
	tx, ok := mid.GetTran(c)
	if !ok {
		return a.hospitalStaffBus, nil
	}

	return a.hospitalStaffBus.NewWithTx(tx)
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
