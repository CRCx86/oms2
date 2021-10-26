package health

import (
	"github.com/gin-gonic/gin"
	"oms2/internal/pkg/service/health"

	"oms2/internal/oms"
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
		apiRoute.POST("/health", c.Health)
	}
}

func (c *Controller) Health(ctx *gin.Context) {
	ctx.Set(oms.KeyResponse, c.service.Health())
}
