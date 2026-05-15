package controllers

import (
	"context"
	"diabetify/internal/services"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var googleTokenVerifier = verifyGoogleToken

var (
	errGoogleTokenVerificationFailed = errors.New("token verification failed")
	errGoogleEmailNotVerified        = errors.New("google account email is not verified")
	errGoogleAudienceMismatch        = errors.New("google token audience does not match configured client")
	errGoogleAudienceMissing         = errors.New("google token audience is missing")
	errGoogleClientIDNotConfigured   = errors.New("google oauth client id is not configured")
	errGoogleIssuerMismatch          = errors.New("google token issuer is invalid")
)

type OauthController struct {
	service services.OAuthService
}

func NewOauthController(service services.OAuthService) *OauthController {
	return &OauthController{service: service}
}

func verifyGoogleToken(ctx context.Context, token string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		"https://oauth2.googleapis.com/tokeninfo?id_token="+token,
		nil,
	)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify token with Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errGoogleTokenVerificationFailed
	}

	var tokenInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	if verified, exists := tokenInfo["email_verified"]; exists {
		switch value := verified.(type) {
		case bool:
			if !value {
				return nil, errGoogleEmailNotVerified
			}
		case string:
			if value != "true" {
				return nil, errGoogleEmailNotVerified
			}
		}
	}

	if err := validateGoogleTokenClaims(tokenInfo); err != nil {
		return nil, err
	}

	return tokenInfo, nil
}

func validateGoogleTokenClaims(tokenInfo map[string]interface{}) error {
	if err := validateGoogleIssuer(tokenInfo); err != nil {
		return err
	}

	if err := validateGoogleAudience(tokenInfo); err != nil {
		return err
	}

	return nil
}

func validateGoogleAudience(tokenInfo map[string]interface{}) error {
	configuredClientIDs := strings.TrimSpace(os.Getenv("GOOGLE_KEY"))
	if configuredClientIDs == "" {
		return errGoogleClientIDNotConfigured
	}

	audience, ok := tokenInfo["aud"].(string)
	if !ok || strings.TrimSpace(audience) == "" {
		return errGoogleAudienceMissing
	}

	for _, clientID := range strings.Split(configuredClientIDs, ",") {
		if strings.TrimSpace(clientID) == audience {
			return nil
		}
	}

	return errGoogleAudienceMismatch
}

func validateGoogleIssuer(tokenInfo map[string]interface{}) error {
	issuer, ok := tokenInfo["iss"].(string)
	if !ok || issuer == "" {
		return nil
	}

	if issuer == "accounts.google.com" || issuer == "https://accounts.google.com" {
		return nil
	}

	return errGoogleIssuerMismatch
}

func (oc *OauthController) GoogleAuth(c *gin.Context) {
	var authRequest struct {
		Token string `json:"token"`
	}

	if err := c.ShouldBindJSON(&authRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Invalid request data",
			"error":   err.Error(),
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	tokenInfo, err := googleTokenVerifier(ctx, authRequest.Token)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errGoogleTokenVerificationFailed) ||
			errors.Is(err, errGoogleEmailNotVerified) ||
			errors.Is(err, errGoogleAudienceMismatch) ||
			errors.Is(err, errGoogleAudienceMissing) ||
			errors.Is(err, errGoogleIssuerMismatch) {
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, gin.H{
			"status":  "error",
			"message": "Failed to verify token with Google",
			"error":   err.Error(),
		})
		return
	}

	if err := validateGoogleTokenClaims(tokenInfo); err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, errGoogleAudienceMismatch) ||
			errors.Is(err, errGoogleAudienceMissing) ||
			errors.Is(err, errGoogleIssuerMismatch) {
			statusCode = http.StatusUnauthorized
		}

		c.JSON(statusCode, gin.H{
			"status":  "error",
			"message": "Failed to verify token with Google",
			"error":   err.Error(),
		})
		return
	}

	email, ok := tokenInfo["email"].(string)
	if !ok || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "Email not found in token",
		})
		return
	}

	name, _ := tokenInfo["name"].(string)
	user, err := oc.service.GetUserByEmail(email)
	isNewUser := err != nil

	if isNewUser {
		user, err = oc.service.CreateGoogleUser(email, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status":  "error",
				"message": "Failed to create user account",
				"error":   err.Error(),
			})
			return
		}
	}

	// Generate JWT
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	})

	jwtSecret := []byte(os.Getenv("JWT_SECRET_KEY"))
	tokenString, err := jwtToken.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": "Could not generate token",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Google authentication successful",
		"data":    tokenString,
	})
}
