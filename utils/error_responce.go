package utils

// ErrorResponse is a struct for error response
type ErrorResponse struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}
