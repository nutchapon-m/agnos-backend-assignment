package staffapp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

type App struct {
	staffBus staffbus.Business
}

func NewApp(staffBus staffbus.Business) *App {
	return &App{staffBus: staffBus}
}

func (a *App) Create(c *gin.Context) {
	var app NewStaff
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	bus, err := a.bus(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	staff, err := bus.Create(c, toNewStaff(app))
	if err != nil {
		if errors.Is(err, staffbus.ErrDuplicate) || errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			response.Fail(c, http.StatusConflict, "Staff Already Exist", staffbus.ErrDuplicate.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppStaff(staff))
}

func (a *App) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	staff, err := a.staffBus.GetByID(c, id)
	if err != nil {
		if errors.Is(err, staffbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Staff Not Found", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAppStaff(staff))
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

	orderBy, err := order.Parse(orderByFields, qp.OrderBy, staffbus.DefaultOrderBy)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	staffs, err := a.staffBus.Query(c, parseFilter(qp), pg, orderBy)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	meta := response.Meta{
		Page:    pg.Number(),
		PerPage: pg.RowsPerPage(),
		Total:   len(staffs),
	}
	response.OKWithMeta(c, toAppStaffs(staffs), meta)
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
		if errors.Is(err, staffbus.ErrNotFound) {
			response.Fail(c, http.StatusNotFound, "Staff Not Found", err.Error())
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
func (a *App) bus(c *gin.Context) (staffbus.Business, error) {
	tx, ok := mid.GetTran(c)
	if !ok {
		return a.staffBus, nil
	}

	return a.staffBus.NewWithTx(tx)
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
