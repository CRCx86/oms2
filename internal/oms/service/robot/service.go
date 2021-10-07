package robot

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/pkg/repository/root"
)

type Service struct {
	zl     *zap.Logger
	cfg    *oms.Config
	root   *root.Repository
	ticker *time.Ticker
	done   chan bool
}

func NewService(cfg *oms.Config, root *root.Repository, zl *zap.Logger) *Service {
	return &Service{
		zl:     zl,
		cfg:    cfg,
		root:   root,
		ticker: time.NewTicker(2 * time.Second),
		done:   make(chan bool),
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
					log.Println(err)
				}
			}
		}
	}()

	go func() {
		select {
		case sig := <-signals:
			log.Printf("Got %s signals. Aborting...", sig)
			s.ticker.Stop()
		}
	}()

	log.Println("Robot has started")

	return nil

}

func (s *Service) Stop(_ context.Context) error {
	s.ticker.Stop()
	s.done <- true
	close(s.done)

	return nil
}

func (s *Service) Do(ctx context.Context, t time.Time) error {

	list, err := s.root.List(ctx)

	s.zl.Sugar().Info(list, err, t)

	return nil

}
