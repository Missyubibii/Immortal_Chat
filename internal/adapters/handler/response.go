// Package handler implements HTTP responses following .rules standard
package handler

// APIResponse represents the standard response envelope
// Per .rules_immortal_chat: ALL API responses must use this format
type APIResponse struct {
	Code    int         `json:"code"`    // HTTP status code (200, 400, 500, etc.)
	Message string      `json:"message"` // Human-readable message ("Success", error description)
	Data    interface{} `json:"data"`    // Actual payload (can be null)
}

// NewSuccessResponse creates a successful response (code 200)
func NewSuccessResponse(data interface{}) APIResponse {
	return APIResponse{
		Code:    200,
		Message: "Success",
		Data:    data,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(code int, message string) APIResponse {
	return APIResponse{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}

// Common error responses
func BadRequestResponse(message string) APIResponse {
	return NewErrorResponse(400, message)
}

func NotFoundResponse(message string) APIResponse {
	return NewErrorResponse(404, message)
}

func InternalErrorResponse(message string) APIResponse {
	return NewErrorResponse(500, message)
}
