package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"

	"diabetify/internal/models"
	"diabetify/internal/repository"

	"github.com/gin-gonic/gin"
)

// pathAction maps (method, fullPath) → action label.
// Only mutating routes we care about are listed here.
var pathAction = map[string]map[string]string{
	"POST": {
		"/data/datasets":                         "UPLOAD_DATASET",
		"/data/datasets/:id/preprocess":          "PREPROCESS_DATASET",
		"/data/training/trigger":                 "TRIGGER_TRAINING",
		"/data/shadow/activate":                  "ACTIVATE_SHADOW",
		"/data/shadow/deactivate/:deployment_id": "DEACTIVATE_SHADOW",
		"/data/drift/trigger":                    "TRIGGER_DRIFT",
		"/expert/models/:id/approve":             "APPROVE_MODEL",
		"/expert/models/:id/reject":              "REJECT_MODEL",
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

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func actionDetails(c *gin.Context, action string, body []byte) string {
	var top map[string]interface{}
	if len(body) > 0 {
		json.Unmarshal(body, &top)
	}

	resp := top
	if nested, ok := top["data"]; ok {
		if m, ok := nested.(map[string]interface{}); ok {
			resp = m
		}
	}

	str := func(key string) string {
		if resp == nil {
			return ""
		}
		if v, ok := resp[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	num := func(key string) int64 {
		if resp == nil {
			return 0
		}
		if v, ok := resp[key]; ok {
			if f, ok := v.(float64); ok {
				return int64(f)
			}
		}
		return 0
	}

	switch action {
	case "UPLOAD_DATASET":
		if fn := str("filename"); fn != "" {
			return fmt.Sprintf("%s (%s)", fn, str("version"))
		}

	case "PREPROCESS_DATASET":
		if fn := str("filename"); fn != "" {
			return fmt.Sprintf("%s (%s)", fn, str("version"))
		}
		return "dataset #" + c.Param("id")

	case "TRIGGER_TRAINING":
		if v := str("version"); v != "" {
			if dv := str("dataset_version"); dv != "" {
				return fmt.Sprintf("model %s dari dataset %s", v, dv)
			}
			return "model " + v
		}

	case "ACTIVATE_SHADOW":
		if mv := str("model_version"); mv != "" {
			return "model " + mv + " diaktifkan"
		}

	case "DEACTIVATE_SHADOW":
		if mv := str("model_version"); mv != "" {
			return "model " + mv + " dihentikan"
		}
		return "deployment " + c.Param("deployment_id") + " dihentikan"

	case "TRIGGER_DRIFT":
		if mv := str("model_version"); mv != "" {
			return fmt.Sprintf("model %s, %d sampel", mv, num("sample_count"))
		}

	case "UPDATE_DRIFT_CONFIG":
		if iv := num("interval_minutes"); iv > 0 {
			return fmt.Sprintf("interval %d mnt, window %d", iv, num("window_size"))
		}

	case "APPROVE_MODEL":
		if v := str("version"); v != "" {
			return "model " + v + " disetujui"
		}
		return "model #" + c.Param("id")

	case "REJECT_MODEL":
		if v := str("version"); v != "" {
			return "model " + v + " ditolak"
		}
		return "model #" + c.Param("id")

	case "UPDATE_USER":
		if e := str("email"); e != "" {
			return e
		}
		return "user #" + c.Param("id")

	case "DELETE_USER":
		return "user #" + c.Param("id")

	case "ACKNOWLEDGE_DRIFT":
		return "alert #" + c.Param("id")
	}
	return ""
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

		responseBody := bw.body.Bytes()

		status := "success"
		var errDetail string
		if bw.Status() >= 400 {
			status = "failed"
			errDetail = extractErrorDetail(responseBody)
		}

		log := &models.AuditLog{
			UserID:      userID,
			Action:      action,
			Details:     actionDetails(c, action, responseBody),
			Status:      status,
			ErrorDetail: errDetail,
		}
		go func() { _ = repo.Create(log) }() //nolint:errcheck
	}
}
