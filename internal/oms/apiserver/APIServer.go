package apiserver

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	stats "github.com/semihalev/gin-stats"
	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/oms/apiserver/middlewares"
	"oms2/internal/pkg/config"
)

type controller interface {
	RegisterRoutes(r *gin.Engine)
}

type APIServer struct {
	zl          *zap.Logger
	Cfg         *config.APIServer
	server      *http.Server
	apiRouter   *gin.Engine
	controllers []controller
}

func (s *APIServer) Start(_ context.Context) error {

	for _, c := range s.controllers {
		c.RegisterRoutes(s.apiRouter)
	}

	addr := net.JoinHostPort(s.Cfg.Host, strconv.Itoa(s.Cfg.ApiPort))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	go func() {
		s.zl.Sugar().Info("server has started on: ", addr)
		if err := s.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}()
	return nil
}

func (s *APIServer) Stop(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, s.Cfg.ShutdownTimeout)
	defer cancel()

	return s.server.Shutdown(c)
}

func (s *APIServer) AddController(c ...controller) *APIServer {
	s.controllers = append(s.controllers, c...)
	return s
}

func NewAPIServer(
	apiServer *config.APIServer,
	cfg *oms.Config,
	l *zap.Logger,
) *APIServer {

	gin.DisableConsoleColor()
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	registerStateMiddlewares(r, l)

	appServer := &http.Server{
		Handler:      r,
		ReadTimeout:  cfg.APIServer.ReadTimeout,
		WriteTimeout: cfg.APIServer.WriteTimeout,
	}

	return &APIServer{
		zl:        l,
		Cfg:       apiServer,
		server:    appServer,
		apiRouter: r,
	}
}

func registerStateMiddlewares(r *gin.Engine, l *zap.Logger) {
	r.Use(stats.RequestStats())
	r.Use(middlewares.RequestMiddleware(l))
	r.Use(middlewares.ResponseMiddleware())
	r.Use(middlewares.RecoveryMiddleware(l))
	r.Use(middlewares.TracerMiddleware())
}
