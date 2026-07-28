package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kar1hsu/backplane/internal/app"
	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/repository"
)

// maxBodyCapture caps how many request/response bytes are stored per log entry,
// so a large payload can't blow up memory or the row.
const maxBodyCapture = 8 << 10 // 8KB

// writeLogTimeout bounds the synchronous log insert so a slow DB can't stall the
// response unwind indefinitely.
const writeLogTimeout = 3 * time.Second

// sensitiveKeys are redacted (value -> ***) wherever they appear in a captured
// JSON request or response body.
var sensitiveKeys = map[string]struct{}{
	"password":         {},
	"old_password":     {},
	"new_password":     {},
	"confirm_password": {},
	"token":            {},
	"secret":           {},
	"access_token":     {},
	"refresh_token":    {},
}

type apiMeta struct {
	module string
	action string
}

// OperationLog records every mutating admin request (POST/PUT/DELETE/PATCH) into
// sys_operation_log. It must sit AFTER AdminAuth (to know the operator) and
// BEFORE CasbinRBAC (so permission-denied attempts are still audited).
//
// Writes are synchronous but best-effort: a logging failure is reported to the
// app log and never fails the business request.
//
// Route → module/action metadata is loaded once from sys_api at startup; APIs
// added at runtime need a restart to enrich their logs (they still log, just
// with empty module/action).
func OperationLog() gin.HandlerFunc {
	repo := repository.NewOperationLogRepo()
	meta := loadAPIMeta()

	return func(c *gin.Context) {
		if !shouldLog(c.Request.Method) {
			c.Next()
			return
		}

		start := time.Now()
		reqParams := captureRequest(c)

		blw := &bodyLogWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = blw

		c.Next()

		m := meta[c.Request.Method+" "+c.FullPath()]
		bizCode, bizMsg := parseBizResult(blw.body.Bytes())
		status := c.Writer.Status()

		entry := &model.SysOperationLog{
			TraceID:    c.GetString("trace_id"),
			UserID:     GetUserID(c),
			Username:   GetUsername(c),
			RoleCodes:  strings.Join(GetRoleCodes(c), ","),
			Module:     m.module,
			Action:     m.action,
			Method:     c.Request.Method,
			Route:      c.FullPath(),
			Path:       c.Request.URL.Path,
			TargetID:   c.Param("id"),
			ReqParams:  reqParams,
			RespParams: captureResponse(blw.body.Bytes()),
			Status:     status,
			BizCode:    bizCode,
			Success:    status == http.StatusOK && bizCode == 0,
			ClientIP:   c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			LatencyMs:  time.Since(start).Milliseconds(),
		}
		if !entry.Success {
			entry.ErrorMsg = bizMsg
		}

		// Decoupled from the request context (which may already be cancelled)
		// and bounded by its own timeout — logging must not block the response.
		ctx, cancel := context.WithTimeout(context.Background(), writeLogTimeout)
		defer cancel()
		if err := repo.Create(ctx, entry); err != nil {
			app.Log.Errorw("write operation log failed", "path", entry.Path, "err", err)
		}
	}
}

// RecordLogin writes an explicit auth event (login/logout). Login lives on a
// public route with no AdminAuth/OperationLog middleware, and a failed login has
// no authenticated user — so auth handlers record it directly. Best-effort.
func RecordLogin(c *gin.Context, action string, userID uint, username string, success bool, errMsg string) {
	entry := &model.SysOperationLog{
		TraceID:   c.GetString("trace_id"),
		UserID:    userID,
		Username:  username,
		Module:    "认证",
		Action:    action,
		Method:    c.Request.Method,
		Route:     c.FullPath(),
		Path:      c.Request.URL.Path,
		Status:    c.Writer.Status(),
		Success:   success,
		ErrorMsg:  errMsg,
		ClientIP:  c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), writeLogTimeout)
	defer cancel()
	if err := repository.NewOperationLogRepo().Create(ctx, entry); err != nil {
		app.Log.Errorw("write login log failed", "username", username, "err", err)
	}
}

func shouldLog(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	default:
		return false
	}
}

// captureRequest reads (capped) the request body, restores it for the handler,
// and returns a JSON string {"query":..., "body":...} with sensitive fields
// redacted. Only JSON / urlencoded bodies are captured (skips file uploads).
func captureRequest(c *gin.Context) string {
	var bodyStr string
	ct := c.ContentType()
	if c.Request.Body != nil && (strings.Contains(ct, "application/json") || strings.Contains(ct, "x-www-form-urlencoded")) {
		buf, _ := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyCapture))
		// Restore: captured prefix + whatever remains unread, so the handler
		// still sees the full body even when it exceeds the capture cap.
		c.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf), c.Request.Body))
		if strings.Contains(ct, "application/json") {
			bodyStr = redactJSON(buf)
		} else {
			bodyStr = string(buf)
		}
	}

	out := make(map[string]string, 2)
	if q := c.Request.URL.RawQuery; q != "" {
		out["query"] = q
	}
	if bodyStr != "" {
		out["body"] = bodyStr
	}
	if len(out) == 0 {
		return ""
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func captureResponse(respBody []byte) string {
	if len(respBody) == 0 {
		return ""
	}

	out := make(map[string]string, 2)
	if len(respBody) > maxBodyCapture {
		out["body"] = "响应内容超过 8KB，已跳过完整记录"
		out["truncated"] = "true"
	} else {
		out["body"] = redactJSON(respBody)
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// redactJSON masks sensitive keys in a JSON object. On parse failure it returns
// the raw bytes as-is (already capped by the caller).
func redactJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	redactValue(v)
	b, err := json.Marshal(v)
	if err != nil {
		return string(raw)
	}
	return string(b)
}

func redactValue(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k := range t {
			if _, ok := sensitiveKeys[strings.ToLower(k)]; ok {
				t[k] = "***"
			} else {
				redactValue(t[k])
			}
		}
	case []interface{}:
		for _, item := range t {
			redactValue(item)
		}
	}
}

// parseBizResult extracts the {code,message} envelope from a response body to
// classify success/failure. Returns (0,"") when the body isn't our envelope.
func parseBizResult(respBody []byte) (int, string) {
	if len(respBody) == 0 {
		return 0, ""
	}
	var r struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(respBody, &r); err != nil {
		return 0, ""
	}
	return r.Code, r.Message
}

func loadAPIMeta() map[string]apiMeta {
	m := make(map[string]apiMeta)
	apis, err := repository.NewApiRepo().ListAll(context.Background())
	if err != nil {
		app.Log.Warnw("operation log: load api meta failed", "err", err)
		return m
	}
	for _, a := range apis {
		m[a.Method+" "+a.Path] = apiMeta{module: a.Group, action: a.Description}
	}
	return m
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}

func (w *bodyLogWriter) WriteString(s string) (int, error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

func (w *bodyLogWriter) capture(b []byte) {
	remaining := maxBodyCapture + 1 - w.body.Len()
	if remaining <= 0 {
		return
	}
	if len(b) > remaining {
		b = b[:remaining]
	}
	w.body.Write(b)
}
