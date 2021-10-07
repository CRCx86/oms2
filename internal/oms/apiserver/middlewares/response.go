package middlewares

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"

	"oms2/internal/oms"
)

func ResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// after request

		if c.Request.Method == http.MethodPost {
			meta := c.MustGet(oms.KeyMeta).(json.RawMessage)
			result, _ := json.Marshal(c.MustGet(oms.KeyResponse))
			var response interface{}

			isError := len(c.Errors) > 0
			if isError {
				stack := strings.Split(string(debug.Stack()), "\n")
				response = oms.ResponseError{
					Success:  0,
					Envelope: oms.Envelope{Meta: meta},
					Error: oms.RError{
						Message:    result,
						StackTrace: stack,
					},
				}
			} else {
				response = oms.ResponseSuccess{
					Success:  1,
					Envelope: oms.Envelope{Meta: meta},
					Data:     result,
				}
			}

			c.JSON(http.StatusOK, response)
		}
	}
}
