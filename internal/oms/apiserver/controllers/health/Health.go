package health

import (
	"github.com/gin-gonic/gin"
	"oms2/internal/oms/service/health"
)

type Controller struct {
	service *health.Service
}

func NewController(service *health.Service) *Controller {
	return &Controller{service: service}
}

func (c *Controller) RegisterRoutes(r *gin.Engine) {

	apiRoute := r.Group("/api")
	{
		apiRoute.GET("/health", c.Health)
		apiRoute.POST("/health", c.Health)
	}
}

func (c *Controller) Health(ctx *gin.Context) {
	resp := c.service.Health()
	ctx.JSON(200, resp)

	ctx.Abort()
}
