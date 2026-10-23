package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

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

// sensitiveKeyParts: a key is redacted (value -> ***) in captured query
// strings and JSON / form bodies when its lowercased name contains any of
// these, covering password, old_password, access_token, client_secret, ...
var sensitiveKeyParts = []string{"password", "passwd", "secret", "token", "authorization", "api_key", "apikey"}

// omittedBody replaces a body that cannot be parsed for redaction (not JSON, or
// cut off at maxBodyCapture). Storing it raw could leak the secrets inside.
const omittedBody = "[内容无法解析或超过 8KB，已省略]"

func isSensitiveKey(key string) bool {
	key = strings.ToLower(key)
	for _, part := range sensitiveKeyParts {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
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
		fitColumns(entry)
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
	fitColumns(entry)
	if err := repository.NewOperationLogRepo().Create(ctx, entry); err != nil {
		app.Log.Errorw("write login log failed", "username", username, "err", err)
	}
}

// fitColumns trims client-controlled fields to their column sizes (see
// model.SysOperationLog). Strict-mode MySQL and PostgreSQL reject an
// over-long value, which would drop the whole audit record — e.g. a login
// attempt with a huge username or a request with a long User-Agent.
func fitColumns(e *model.SysOperationLog) {
	e.TraceID = truncateRunes(e.TraceID, 32)
	e.Username = truncateRunes(e.Username, 64)
	e.RoleCodes = truncateRunes(e.RoleCodes, 255)
	e.Module = truncateRunes(e.Module, 64)
	e.Action = truncateRunes(e.Action, 128)
	e.Method = truncateRunes(e.Method, 16)
	e.Route = truncateRunes(e.Route, 255)
	e.Path = truncateRunes(e.Path, 255)
	e.TargetID = truncateRunes(e.TargetID, 64)
	e.ClientIP = truncateRunes(e.ClientIP, 64)
	e.UserAgent = truncateRunes(e.UserAgent, 255)
}

// truncateRunes cuts s to at most n characters (varchar sizes count
// characters, and cutting bytes could split a multi-byte character).
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
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
			bodyStr = redactForm(string(buf))
		}
	}

	out := make(map[string]string, 2)
	if q := c.Request.URL.RawQuery; q != "" {
		out["query"] = redactForm(q)
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

// redactJSON masks sensitive keys in a JSON document. A body that does not
// parse — including JSON cut off at maxBodyCapture — is omitted, not stored raw.
func redactJSON(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return omittedBody
	}
	redactValue(v)
	b, err := json.Marshal(v)
	if err != nil {
		return omittedBody
	}
	return string(b)
}

// redactForm masks sensitive keys in a query string or urlencoded body.
func redactForm(raw string) string {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return omittedBody
	}
	for k := range values {
		if isSensitiveKey(k) {
			values[k] = []string{"***"}
		}
	}
	return values.Encode()
}

func redactValue(v interface{}) {
	switch t := v.(type) {
	case map[string]interface{}:
		for k := range t {
			if isSensitiveKey(k) {
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
