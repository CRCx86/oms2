package robot

import (
	"context"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/pkg/repository/robot"
)

const (
	IterationModel = "Iteration"
	TilingModel    = "Tiling"
)

type Service struct {
	zl              *zap.Logger
	cfg             *oms.Config
	ticker          *time.Ticker
	done            chan bool
	model           string
	action          *Action
	robotRepository *robot.Repository
}

func NewService(cfg *oms.Config, action *Action, r *robot.Repository, zl *zap.Logger) *Service {
	return &Service{
		zl:              zl,
		cfg:             cfg,
		ticker:          time.NewTicker(2 * time.Second),
		done:            make(chan bool),
		model:           IterationModel,
		action:          action,
		robotRepository: r,
	}
}

func (s *Service) Start(ctx context.Context) error {

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)

	go func() {

		for {
			select {
			case <-s.done:
				return
			case t := <-s.ticker.C:
				err := s.Do(ctx, t)
				if err != nil {
					s.ticker.Stop()
					s.done <- true
					s.zl.Sugar().Error(err)
				}
			}
		}
	}()

	go func() {
		select {
		case sig := <-signals:
			s.zl.Sugar().Info("Got %s signals. Aborting...", sig)
			s.ticker.Stop()
		}
	}()

	s.zl.Sugar().Info("Robot has started")

	return nil

}

func (s *Service) Stop(_ context.Context) error {
	s.ticker.Stop()
	s.done <- true
	close(s.done)

	return nil
}

func (s *Service) Do(ctx context.Context, t time.Time) (err error) {

	if s.model == TilingModel {
		err = s.Tiling(ctx, t)
	} else {
		err = s.Iteration(ctx, t)
	}

	return err
}

func (s *Service) Iteration(ctx context.Context, t time.Time) (ok error) {

	ok = s.DoStep(ctx)
	if ok != nil {
		s.zl.Sugar().Error(ok)
		return ok
	}

	s.zl.Sugar().Info("Robot at", t)
	return ok
}

func (s *Service) Tiling(ctx context.Context, t time.Time) (ok error) {

	return nil
}

func (s *Service) DoStep(ctx context.Context) (ok error) {

	return ok
}

func (s *Service) DoStepAndEvents() (result interface{}) {

	//result = a.DoIncomingEvents()
	//result = a.DoNextStep(nil)
	return result
}
