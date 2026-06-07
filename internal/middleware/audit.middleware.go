package middleware

import (
	"bytes"
	"encoding/json"

	"diabetify/internal/models"
	"diabetify/internal/repository"

	"github.com/gin-gonic/gin"
)

// pathAction maps (method, fullPath) → action label.
// Only mutating routes we care about are listed here.
var pathAction = map[string]map[string]string{
	"POST": {
		"/data/datasets":                          "UPLOAD_DATASET",
		"/data/datasets/:id/preprocess":           "PREPROCESS_DATASET",
		"/data/training/trigger":                  "TRIGGER_TRAINING",
		"/data/shadow/activate":                   "ACTIVATE_SHADOW",
		"/data/shadow/deactivate/:deployment_id":  "DEACTIVATE_SHADOW",
		"/data/drift/trigger":                     "TRIGGER_DRIFT",
		"/data/drift/alerts/:id/acknowledge":      "ACKNOWLEDGE_DRIFT",
		"/expert/models/:id/approve":              "APPROVE_MODEL",
		"/expert/models/:id/reject":               "REJECT_MODEL",
	},
	"PUT": {
		"/data/drift/config": "UPDATE_DRIFT_CONFIG",
	},
	"PATCH": {
		"/admin/users/:id": "UPDATE_USER",
	},
	"DELETE": {
		"/admin/users/:id": "DELETE_USER",
	},
}

// bodyWriter wraps gin.ResponseWriter to capture response body for error logging.
type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func actionDetails(c *gin.Context, action string) string {
	switch action {
	case "PREPROCESS_DATASET":
		return "dataset #" + c.Param("id")
	case "DEACTIVATE_SHADOW":
		return "deployment #" + c.Param("deployment_id")
	case "ACKNOWLEDGE_DRIFT":
		return "alert #" + c.Param("id")
	case "APPROVE_MODEL", "REJECT_MODEL":
		return "model #" + c.Param("id")
	case "UPDATE_USER", "DELETE_USER":
		return "user #" + c.Param("id")
	default:
		return ""
	}
}

func extractErrorDetail(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var errBody struct {
		Detail string `json:"detail"`
	}
	if err := json.Unmarshal(body, &errBody); err == nil && errBody.Detail != "" {
		if len(errBody.Detail) > 1000 {
			return errBody.Detail[:1000]
		}
		return errBody.Detail
	}
	raw := string(body)
	if len(raw) > 1000 {
		raw = raw[:1000]
	}
	return raw
}

// AuditMiddleware logs every auditable mutating request after the handler runs.
// It requires AuthMiddleware to have set "user_id" in context first.
func AuditMiddleware(repo repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only wrap body capture for auditable paths to avoid unnecessary buffering.
		methods, isAuditable := pathAction[c.Request.Method]
		var bw *bodyWriter
		if isAuditable {
			if _, found := methods[c.FullPath()]; found {
				bw = &bodyWriter{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
				c.Writer = bw
			}
		}

		c.Next()

		if bw == nil {
			return
		}

		userID := c.GetUint("user_id")
		if userID == 0 {
			return
		}

		action, ok := methods[c.FullPath()]
		if !ok {
			return
		}

		status := "success"
		var errDetail string
		if bw.Status() >= 400 {
			status = "failed"
			errDetail = extractErrorDetail(bw.body.Bytes())
		}

		log := &models.AuditLog{
			UserID:      userID,
			Action:      action,
			Details:     actionDetails(c, action),
			Status:      status,
			ErrorDetail: errDetail,
		}
		go func() { _ = repo.Create(log) }() //nolint:errcheck
	}
}
