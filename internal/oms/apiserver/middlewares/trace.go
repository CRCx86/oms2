package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/opentracing/opentracing-go"
)

func TracerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		span := opentracing.GlobalTracer().StartSpan(c.Request.URL.Path)
		defer span.Finish()

		c.Set("span", span)

		c.Next()
	}
}
