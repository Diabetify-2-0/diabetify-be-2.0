package httpx

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func BindJSONStrict(c *gin.Context, dst interface{}) error {
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if decoder.More() {
		return fmt.Errorf("request body must contain a single JSON object")
	}
	return nil
}

func Error(c *gin.Context, statusCode int, message string, err error) {
	payload := gin.H{
		"status":  "error",
		"message": message,
	}
	if err != nil {
		payload["error"] = err.Error()
	}
	c.JSON(statusCode, payload)
}

func ErrorText(c *gin.Context, statusCode int, message, errorText string) {
	payload := gin.H{
		"status":  "error",
		"message": message,
	}
	if errorText != "" {
		payload["error"] = errorText
	}
	c.JSON(statusCode, payload)
}

func Unauthorized(c *gin.Context) {
	ErrorText(c, http.StatusUnauthorized, "Unauthorized access", "")
}
