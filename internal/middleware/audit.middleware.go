package middleware

import (
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
		"/data/training/jobs/:id/cancel":          "CANCEL_TRAINING",
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

func actionDetails(c *gin.Context, action string) string {
	switch action {
	case "PREPROCESS_DATASET":
		return "dataset #" + c.Param("id")
	case "CANCEL_TRAINING":
		return "job #" + c.Param("id")
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

// AuditMiddleware logs every auditable mutating request after the handler runs.
// It requires AuthMiddleware to have set "user_id" in context first.
func AuditMiddleware(repo repository.AuditLogRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		userID := c.GetUint("user_id")
		if userID == 0 {
			return
		}

		methods, ok := pathAction[c.Request.Method]
		if !ok {
			return
		}
		action, ok := methods[c.FullPath()]
		if !ok {
			return
		}

		status := "success"
		if c.Writer.Status() >= 400 {
			status = "failed"
		}

		log := &models.AuditLog{
			UserID:  userID,
			Action:  action,
			Details: actionDetails(c, action),
			Status:  status,
		}
		go func() { _ = repo.Create(log) }() //nolint:errcheck
	}
}
