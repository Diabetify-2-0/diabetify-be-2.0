package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"diabetify/internal/repository"

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
func MLOpsProxy(targetPath string, userRepo repository.UserRepository) gin.HandlerFunc {
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

		if ct := c.GetHeader("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}

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

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to read MLOps response",
				"error":   err.Error(),
			})
			return
		}

		if c.Request.Method == "GET" && strings.HasSuffix(path, "/datasets") && resp.StatusCode == http.StatusOK {
			var response map[string]interface{}
			if err := json.Unmarshal(body, &response); err == nil {
				if data, ok := response["data"].([]interface{}); ok {
					for _, item := range data {
						if dataset, ok := item.(map[string]interface{}); ok {
							if uploadedByVal, ok := dataset["uploaded_by"]; ok {
								uploadedByInt, _ := uploadedByVal.(float64)
								uploaderID := uint(uploadedByInt)

								user, err := userRepo.GetUserByID(uploaderID)
								if err == nil && user != nil {
									dataset["uploader_name"] = user.Name
								} else {
									dataset["uploader_name"] = fmt.Sprintf("User #%d", uploaderID)
								}
							}
						}
					}

					enrichedBody, err := json.Marshal(response)
					if err == nil {
						body = enrichedBody
					}
				}
			}
		}

		for key, values := range resp.Header {
			if key == "Content-Length" || key == "Content-Encoding" || key == "Transfer-Encoding" {
				continue
			}
			for _, v := range values {
				c.Header(key, v)
			}
		}

		c.Status(resp.StatusCode)
		c.Data(resp.StatusCode, "application/json", body)
	}
}
