package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"diabetify/internal/config"
	"diabetify/internal/repository"

	"github.com/gin-gonic/gin"
)

var hopByHopHeaders = map[string]bool{
	"Connection":          true,
	"Keep-Alive":          true,
	"Proxy-Authenticate":  true,
	"Proxy-Authorization": true,
	"Te":                  true,
	"Trailer":             true,
	"Transfer-Encoding":   true,
	"Upgrade":             true,
}

var mlopsHTTPClient = &http.Client{
	Timeout: 120 * time.Second, // generous timeout for file uploads and synchronous training
}

func getMLOpsBaseURL() string {
	return config.Load().MLOps.ServiceURL
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

		// Only buffer for endpoints where we need to enrich user names.
		// All other responses (including file downloads) are streamed directly.
		needsEnrichment := c.Request.Method == "GET" && resp.StatusCode == http.StatusOK &&
			(strings.HasSuffix(path, "/datasets") || strings.HasSuffix(path, "/training/jobs") || strings.HasSuffix(path, "/models"))

		if !needsEnrichment {
			for key, values := range resp.Header {
				if hopByHopHeaders[key] {
					continue
				}
				for _, v := range values {
					c.Header(key, v)
				}
			}
			c.Status(resp.StatusCode)
			io.Copy(c.Writer, resp.Body) //nolint:errcheck
			return
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to read MLOps response",
				"error":   err.Error(),
			})
			return
		}

		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err == nil {
			if data, ok := response["data"].([]interface{}); ok {
				uniqueIDs := make(map[uint]bool)
				for _, item := range data {
					if row, ok := item.(map[string]interface{}); ok {
						for _, key := range []string{"uploaded_by", "triggered_by", "trained_by"} {
							if val, ok := row[key]; ok {
								if f, ok := val.(float64); ok {
									uniqueIDs[uint(f)] = true
								}
							}
						}
					}
				}

				userNames := make(map[uint]string, len(uniqueIDs))
				for id := range uniqueIDs {
					if user, err := userRepo.GetUserByID(id); err == nil && user != nil {
						userNames[id] = user.Name
					} else {
						userNames[id] = fmt.Sprintf("User #%d", id)
					}
				}

				for _, item := range data {
					if row, ok := item.(map[string]interface{}); ok {
						if val, ok := row["uploaded_by"]; ok {
							if f, ok := val.(float64); ok {
								row["uploader_name"] = userNames[uint(f)]
							}
						}
						if val, ok := row["triggered_by"]; ok {
							if f, ok := val.(float64); ok {
								row["triggered_by_name"] = userNames[uint(f)]
							}
						}
						if val, ok := row["trained_by"]; ok {
							if f, ok := val.(float64); ok {
								row["trained_by_name"] = userNames[uint(f)]
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

		for key, values := range resp.Header {
			if hopByHopHeaders[key] || key == "Content-Length" || key == "Content-Encoding" {
				continue
			}
			for _, v := range values {
				c.Header(key, v)
			}
		}

		c.Data(resp.StatusCode, "application/json", body)
	}
}
