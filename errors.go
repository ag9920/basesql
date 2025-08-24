package basesql

import (
	"errors"
	"fmt"
)

// 预定义错误
var (
	ErrConnectionFailed   = errors.New("basesql: connection failed")
	ErrInvalidCredentials = errors.New("basesql: invalid credentials")
	ErrTableNotFound      = errors.New("basesql: table not found")
	ErrFieldNotFound      = errors.New("basesql: field not found")
	ErrRecordNotFound     = errors.New("basesql: record not found")
	ErrUnsupportedType    = errors.New("basesql: unsupported data type")
	ErrRateLimitExceeded  = errors.New("basesql: rate limit exceeded")
	ErrBatchSizeExceeded  = errors.New("basesql: batch size exceeded")
	ErrInvalidQuery       = errors.New("basesql: invalid query")
	ErrPermissionDenied   = errors.New("basesql: permission denied")
	ErrInvalidOperation   = errors.New("basesql: invalid operation")
)

// BaseError 基础错误类型
type BaseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *BaseError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("basesql [%s]: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("basesql [%s]: %s", e.Code, e.Message)
}

// 错误构造函数
func ErrInvalidConfig(details string) error {
	return &BaseError{
		Code:    "INVALID_CONFIG",
		Message: "invalid configuration",
		Details: details,
	}
}

func ErrAPICall(details string) error {
	return &BaseError{
		Code:    "API_CALL_FAILED",
		Message: "API call failed",
		Details: details,
	}
}

func ErrAuth(details string) error {
	return &BaseError{
		Code:    "AUTH_FAILED",
		Message: "authentication failed",
		Details: details,
	}
}

func ErrDataMapping(details string) error {
	return &BaseError{
		Code:    "DATA_MAPPING_FAILED",
		Message: "data mapping failed",
		Details: details,
	}
}

func ErrSQLParsing(details string) error {
	return &BaseError{
		Code:    "SQL_PARSING_FAILED",
		Message: "SQL parsing failed",
		Details: details,
	}
}

// IsPermissionError 判断是否为权限错误
func IsPermissionError(err error) bool {
	if err == nil {
		return false
	}

	switch err {
	case ErrInvalidCredentials, ErrPermissionDenied:
		return true
	}

	if baseErr, ok := err.(*BaseError); ok {
		switch baseErr.Code {
		case "AUTH_FAILED", "PERMISSION_DENIED":
			return true
		}
	}

	return false
}
