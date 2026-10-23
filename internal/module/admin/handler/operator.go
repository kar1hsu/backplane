package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/middleware"
	"github.com/kar1hsu/backplane/internal/module/admin/service"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
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
