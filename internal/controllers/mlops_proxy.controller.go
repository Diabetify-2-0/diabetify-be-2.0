package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var mlopsHTTPClient = &http.Client{
	Timeout: 120 * time.Second, // generous timeout for file uploads and synchronous training
}

func getMLOpsBaseURL() string {
	url := os.Getenv("MLOPS_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8000"
	}
	return strings.TrimRight(url, "/")
}

// MLOpsProxy returns a Gin handler that proxies the current request to the MLOps service.
// targetPath supports Gin-style params (e.g. /datasets/:id) that are resolved from c.Params.
func MLOpsProxy(targetPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")

		// Resolve Gin path params into the target path
		path := targetPath
		for _, param := range c.Params {
			path = strings.ReplaceAll(path, ":"+param.Key, param.Value)
		}

		targetURL := getMLOpsBaseURL() + path
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		req, err := http.NewRequestWithContext(
			c.Request.Context(),
			c.Request.Method,
			targetURL,
			c.Request.Body,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to build proxy request",
				"error":   err.Error(),
			})
			return
		}

		// Preserve Content-Type (critical for multipart/form-data boundary)
		if ct := c.GetHeader("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}

		// Inject authenticated user ID so MLOps can record uploaded_by / triggered_by
		req.Header.Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))

		resp, err := mlopsHTTPClient.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{
				"status":  "error",
				"message": fmt.Sprintf("MLOps service unreachable: %v", err),
			})
			return
		}
		defer resp.Body.Close()

		// Forward response headers
		for key, values := range resp.Header {
			for _, v := range values {
				c.Header(key, v)
			}
		}

		c.Status(resp.StatusCode)
		io.Copy(c.Writer, resp.Body) //nolint:errcheck
	}
}
