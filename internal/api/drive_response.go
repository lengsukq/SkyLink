package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type driveErrorCode string

const (
	driveErrUnauthorized   driveErrorCode = "unauthorized"
	driveErrInvalidBody    driveErrorCode = "invalid_body"
	driveErrInvalidPath    driveErrorCode = "invalid_path"
	driveErrNotFound       driveErrorCode = "not_found"
	driveErrQuotaExceeded  driveErrorCode = "quota_exceeded"
	driveErrInvalidRequest driveErrorCode = "invalid_request"
	driveErrInternal       driveErrorCode = "internal_error"
)

func driveOK(c *gin.Context, payload gin.H) {
	if payload == nil {
		payload = gin.H{}
	}
	payload["ok"] = true
	c.JSON(http.StatusOK, payload)
}

// driveError keeps backward compatibility with the frontend by still returning "error": "<message>".
func driveError(c *gin.Context, status int, code driveErrorCode, message string) {
	if status <= 0 {
		status = http.StatusInternalServerError
	}
	c.JSON(status, gin.H{
		"ok":    false,
		"code":  code,
		"error": message,
	})
}

