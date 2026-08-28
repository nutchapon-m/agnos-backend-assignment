package staffapp

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/mid"
	"github.com/nutchapon-m/agnos-backend-assignment/src/app/sdk/response"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/hospitalstaffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/staffbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/domain/userbus"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/order"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/page"
	"github.com/nutchapon-m/agnos-backend-assignment/src/business/sdk/sqldb"
)

var (
	// errInvalidCredentials is the single answer to every failed login, whatever
	// the real reason was.
	errInvalidCredentials = errors.New("invalid username or password")
	errStaffInactive      = errors.New("staff account is not active")
	errNotAssigned        = errors.New("staff is not assigned to this hospital")
)

type App struct {
	hospitalStaffBus hospitalstaffbus.Business
	staffBus         staffbus.Business
	userBus          userbus.Business
}

func NewApp(hospitalStaffBus hospitalstaffbus.Business, staffBus staffbus.Business, userBus userbus.Business) *App {
	return &App{
		hospitalStaffBus: hospitalStaffBus,
		staffBus:         staffBus,
		userBus:          userBus,
	}
}

func (a *App) Create(c *gin.Context) {
	var app NewRegistration
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	userBus, staffBus, hospitalStaffBus, err := a.buses(c)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	user, err := userBus.Create(c, toNewUser(app))
	if err != nil {
		switch {
		case errors.Is(err, userbus.ErrDuplicate), errors.Is(err, sqldb.ErrDBDuplicatedEntry):
			response.Fail(c, http.StatusConflict, "User Already Exist", userbus.ErrDuplicate.Error())
		case errors.Is(err, userbus.ErrInvalidPassword):
			response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	staff, err := staffBus.Create(c, toNewStaff(app, user.ID))
	if err != nil {
		if errors.Is(err, staffbus.ErrDuplicate) || errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			response.Fail(c, http.StatusConflict, "Staff Already Exist", staffbus.ErrDuplicate.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	hospitalStaff, err := hospitalStaffBus.RegisterStaff(c, app.HospitalID, staff.ID)
	if err != nil {
		switch {
		case errors.Is(err, hospitalstaffbus.ErrDuplicate):
			response.Fail(c, http.StatusConflict, "Staff Already Assigned", err.Error())
		case errors.Is(err, hospitalstaffbus.ErrInvalidReference):
			response.Fail(c, http.StatusNotFound, "Hospital Not Found", err.Error())
		default:
			response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		}
		return
	}

	response.OK(c, toRegistration(staff, hospitalStaff))
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

// Login authenticates a staff member against one hospital. It answers 401 for
// anything that comes down to "these credentials are not a staff account" -
// unknown username, wrong password, or a user with no staff record - so the
// response cannot be used to find out which usernames exist.
func (a *App) Login(c *gin.Context) {
	var app LoginStaff
	if err := c.ShouldBindJSON(&app); err != nil {
		response.Fail(c, http.StatusBadRequest, "Invalid Argument", err.Error())
		return
	}

	user, err := a.userBus.Authenticate(c, app.Username, app.Password)
	if err != nil {
		if errors.Is(err, userbus.ErrAuthenticationFailure) {
			response.Fail(c, http.StatusUnauthorized, "Unauthorized", errInvalidCredentials.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	staff, err := a.staffOf(c, user.ID)
	if err != nil {
		if errors.Is(err, errInvalidCredentials) {
			response.Fail(c, http.StatusUnauthorized, "Unauthorized", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	if !staff.IsActive {
		response.Fail(c, http.StatusForbidden, "Staff Inactive", errStaffInactive.Error())
		return
	}

	hospitalStaff, err := a.assignment(c, staff.ID, app.HospitalID)
	if err != nil {
		if errors.Is(err, errNotAssigned) {
			response.Fail(c, http.StatusForbidden, "Hospital Not Allowed", err.Error())
			return
		}
		response.Fail(c, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	response.OK(c, toAuthentication(staff, hospitalStaff))
}

// staffOf returns the staff record the user owns. A user without one is not a
// staff member and is reported as bad credentials.
func (a *App) staffOf(c *gin.Context, userID int) (staffbus.Staff, error) {
	filter := staffbus.QueryFilter{UserID: &userID}

	staffs, err := a.staffBus.Query(c, filter, page.MustParse(1, 1), staffbus.DefaultOrderBy)
	if err != nil {
		return staffbus.Staff{}, err
	}

	if len(staffs) == 0 {
		return staffbus.Staff{}, errInvalidCredentials
	}

	return staffs[0], nil
}

// assignment returns the staff member's active assignment to the hospital, or
// errNotAssigned when there is none.
func (a *App) assignment(c *gin.Context, staffID, hospitalID int) (hospitalstaffbus.HospitalStaff, error) {
	active := true
	filter := hospitalstaffbus.QueryFilter{
		StaffID:    &staffID,
		HospitalID: &hospitalID,
		Active:     &active,
	}

	assignments, err := a.hospitalStaffBus.Query(c, filter, page.MustParse(1, 1), hospitalstaffbus.DefaultOrderBy)
	if err != nil {
		return hospitalstaffbus.HospitalStaff{}, err
	}

	if len(assignments) == 0 {
		return hospitalstaffbus.HospitalStaff{}, errNotAssigned
	}

	return assignments[0], nil
}

// buses returns the three businesses the create flow writes through, scoped to
// the request transaction when the BeginCommitRollback middleware is in play.
// They must share one transaction or a half-finished registration would be
// committed.
func (a *App) buses(c *gin.Context) (userbus.Business, staffbus.Business, hospitalstaffbus.Business, error) {
	tx, ok := mid.GetTran(c)
	if !ok {
		return a.userBus, a.staffBus, a.hospitalStaffBus, nil
	}

	userBus, err := a.userBus.NewWithTx(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	staffBus, err := a.staffBus.NewWithTx(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	hospitalStaffBus, err := a.hospitalStaffBus.NewWithTx(tx)
	if err != nil {
		return nil, nil, nil, err
	}

	return userBus, staffBus, hospitalStaffBus, nil
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
