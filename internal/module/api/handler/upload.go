package handler

import (
	"errors"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/pkg/errcode"
	"github.com/kar1hsu/backplane/internal/pkg/response"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/pkg/storage"
)

const (
	bytesPerMegabyte       int64 = 1024 * 1024
	multipartOverheadBytes int64 = 1024 * 1024
)

type fileUploader interface {
	Save(file *multipart.FileHeader, folders ...string) (*storage.UploadedFile, error)
}

type UploadHandler struct {
	uploader          fileUploader
	getResourceDomain func() string
}

type UploadResult struct {
	*storage.UploadedFile
	ResourceDomain string `json:"resource_domain"`
}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{
		uploader: app.Uploader,
		getResourceDomain: func() string {
			return setting.GetString(setting.ResourceDomainKey)
		},
	}
}

func (h *UploadHandler) Upload(c *gin.Context) {
	if app.Cfg.Storage.MaxSize > 0 {
		maxBodyBytes := app.Cfg.Storage.MaxSize*bytesPerMegabyte + multipartOverheadBytes
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)
	}

	file, err := c.FormFile("file")
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			response.Fail(c, errcode.ErrParam, "文件大小超过限制")
			return
		}
		response.Fail(c, errcode.ErrParam, "请选择上传文件")
		return
	}

	if h.uploader == nil {
		response.Fail(c, errcode.ErrServer, "文件存储服务未初始化")
		return
	}

	result, err := h.uploader.Save(file, c.PostForm("folder"))
	switch {
	case err == nil:
		resourceDomain := ""
		if h.getResourceDomain != nil {
			resourceDomain = strings.TrimRight(strings.TrimSpace(h.getResourceDomain()), "/")
		}
		response.OK(c, &UploadResult{
			UploadedFile:   result,
			ResourceDomain: resourceDomain,
		})
	case errors.Is(err, storage.ErrEmptyFile):
		response.Fail(c, errcode.ErrParam, "上传文件不能为空")
	case errors.Is(err, storage.ErrFileTooLarge):
		response.Fail(c, errcode.ErrParam, "文件大小超过限制")
	case errors.Is(err, storage.ErrInvalidFileType):
		response.Fail(c, errcode.ErrParam, "不支持的文件类型")
	case errors.Is(err, storage.ErrInvalidFolder):
		response.Fail(c, errcode.ErrParam, "文件夹名称不合法")
	default:
		if app.Log != nil {
			app.Log.Errorw("upload file failed", "err", err)
		}
		response.Fail(c, errcode.ErrServer, "文件上传失败")
	}
}
