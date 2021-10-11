package robot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/pkg/repository/robot"
)

const (
	IterationModel = "Iteration"
	TilingModel    = "Tiling"
)

const (
	action    = "action"
	wait      = "wait"
	terminate = "terminate"
)

type Service struct {
	zl              *zap.Logger
	cfg             *oms.Config
	ticker          *time.Ticker
	done            chan bool
	model           string
	action          *Action
	robotRepository *robot.Repository
	wg              sync.WaitGroup
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

	s.wg.Add(1)

	go func() {

		defer s.wg.Done()

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
	s.wg.Wait()

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

	// TODO: designed

	return nil
}

func (s *Service) DoStep(ctx context.Context) (ok error) {

	processing, ok := s.robotRepository.Processing(ctx)
	fmt.Println(len(processing), "PROCESSING LEN:")
	if ok != nil {
		s.zl.Sugar().Error(ok)
		return ok
	}

	for _, item := range processing {
		go func(data map[string]interface{}) {
			err := s.DoStepAndEvents(ctx, data)
			if err != nil {
				s.zl.Sugar().Info(err)
			}
		}(item)
	}

	return ok
}

func (s *Service) DoStepAndEvents(ctx context.Context, data map[string]interface{}) (result interface{}) {

	result = s.DoIncomingEvents(ctx)
	if result != nil {
		return result
	}

	return s.DoNextStep(ctx, data)
}

func (s *Service) DoIncomingEvents(ctx context.Context) error {

	events, ok := s.robotRepository.FindEventsPerStep(ctx)
	if ok != nil {
		s.zl.Sugar().Error(ok)
		return ok
	}

	for _, event := range events {
		ok := s.RecordToNextStep(ctx, event)
		if ok != nil {
			return ok
		}
	}

	return ok
}

func (s *Service) RecordToNextStep(ctx context.Context, event map[string]interface{}) (ok error) {

	updated, ok := s.robotRepository.UpdateProcessing(ctx, event)
	s.zl.Sugar().Info(event, updated)
	if ok != nil {
		s.zl.Sugar().Error(ok)
	}

	return ok
}

func (s *Service) DoNextStep(ctx context.Context, data map[string]interface{}) error {

	t := data["type"]
	if t == action {
		_action := data["action"]
		err := s.DoAction(_action, data)
		if err == nil {
			err = s.StepToNextNode(ctx, data)
			if err != nil {
				return err
			}
		}
	} else if t == wait {

		w := data["waiting_time"].(int32)
		e := data["entry_time"].(time.Time)

		leaveSeconds := time.Now().Second() + int(w)
		timeLine := time.Unix(int64(leaveSeconds), 0)
		if e.Before(timeLine) {
			err := s.StepToNextNode(ctx, data)
			if err != nil {
				return err
			}
		}
	} else if t == terminate {
		return s.Terminate(ctx, data)
	}

	return nil
}

func (s *Service) DoAction(a interface{}, data map[string]interface{}) error {
	return s.InvokeAction(a, data)
}

func (s *Service) StepToNextNode(ctx context.Context, data map[string]interface{}) error {

	nextNode, ok := s.FindNextNode(ctx, data["node_id"])
	if ok == nil && nextNode != 0 {
		s.zl.Sugar().Warn("Next step...")
		return s.RecordToNextStep(ctx, data)
	}

	return ok
}

func (s *Service) FindNextNode(ctx context.Context, i interface{}) (uint, error) {

	_sql, args, err := squirrel.StatementBuilder.
		Select("*").
		From("nodes").
		Where(squirrel.Gt{"id": i}).
		OrderBy("id").
		Limit(1).
		ToSql()

	if err != nil {
		return 0, err
	}

	results, err := s.robotRepository.RootRepository.Get(ctx, _sql, args)
	if err != nil {
		return 0, err
	}

	for _, result := range results {
		return *result["id"].(*uint), nil
	}

	return 0, nil

}

func (s *Service) Terminate(ctx context.Context, data map[string]interface{}) error {

	_sql, args, err := squirrel.
		StatementBuilder.
		Delete("lots_nodes").
		Where(squirrel.Eq{"id": data["id"]}).
		ToSql()
	if err != nil {
		return err
	}

	return s.robotRepository.RootRepository.Delete(ctx, _sql, args...)
}

func (s *Service) InvokeAction(action interface{}, data map[string]interface{}) error {

	var t Action
	var args []interface{}
	args = append(args, data)

	name := action.(string)

	inputs := make([]reflect.Value, len(args))
	for i, _ := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}
	values := reflect.ValueOf(&t).MethodByName(name).Call(inputs)
	err := values[0].Interface()
	if err != nil {
		return err.(error)
	}

	return nil
}
