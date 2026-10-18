package response

import "github.com/gin-gonic/gin"

type Envelope struct {
	Success bool       `json:"success"`
	Message string     `json:"message"`
	Data    any        `json:"data"`
	Error   *ErrorBody `json:"error"`
}
type ErrorBody struct {
	Code    string `json:"code"`
	Details any    `json:"details"`
}

func OK(c *gin.Context, status int, message string, data any) {
	c.JSON(status, Envelope{Success: true, Message: message, Data: data, Error: nil})
}
func Fail(c *gin.Context, status int, code, message string, details any) {
	c.JSON(status, Envelope{Success: false, Message: message, Data: nil, Error: &ErrorBody{Code: code, Details: details}})
}
