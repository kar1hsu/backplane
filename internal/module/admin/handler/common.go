package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"gorm.io/gorm"
)

// operatorFrom identifies the authenticated admin; AdminAuth has already
// loaded their enabled role codes from the DB.
func operatorFrom(c *gin.Context) service.Operator {
	return service.Operator{
		UserID:    middleware.GetUserID(c),
		RoleCodes: middleware.GetRoleCodes(c),
	}
}

// failWith answers a service error: permission rule violations become
// errcode.ErrForbidden, everything else errcode.ErrServer.
func failWith(c *gin.Context, err error) {
	if service.IsForbidden(err) {
		response.Fail(c, errcode.ErrForbidden, err.Error())
		return
	}
	response.Fail(c, errcode.ErrServer, err.Error())
}

// failLookup answers a GetByID error: a missing record keeps the resource's
// not-found code, while a real DB failure is logged and reported as a server
// error instead of being disguised as "not found".
func failLookup(c *gin.Context, err error, notFoundCode int, notFoundMsg string) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.Fail(c, notFoundCode, notFoundMsg)
		return
	}
	app.Log.Errorw("lookup failed", "route", c.FullPath(), "err", err)
	response.Fail(c, errcode.ErrServer, "服务器内部错误")
}
