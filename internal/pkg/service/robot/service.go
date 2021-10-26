package robot

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"runtime/pprof"
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
	zl     *zap.Logger
	cfg    *oms.Config
	ticker *time.Ticker

	done   chan bool
	model  string
	action *Action

	robotRepository *robot.Repository
	wg              sync.WaitGroup
}

func NewService(cfg *oms.Config, action *Action, r *robot.Repository, zl *zap.Logger) *Service {
	return &Service{
		zl:              zl,
		cfg:             cfg,
		ticker:          time.NewTicker(1 * time.Second),
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
				s.ticker.Stop()
				return
			case t := <-s.ticker.C:
				c, cancel := context.WithTimeout(context.Background(), s.cfg.MaxCollectTime)
				err := s.Do(c, t)
				if err != nil {
					s.ticker.Stop()
					s.done <- true
					s.zl.Sugar().Error(err)
					cancel()
				}
				cancel()
			}
		}
	}()

	go func() {
		select {
		case <-ctx.Done():
			return
		case sig := <-signals:
			s.zl.Sugar().Info(fmt.Sprintf("Got %s signals. Aborting...", sig))
			s.done <- true
		}
	}()

	s.zl.Sugar().Info("Robot has started")

	return nil

}

func (s *Service) Stop(_ context.Context) error {

	s.done <- true
	close(s.done)

	s.wg.Wait()

	return nil
}

func (s *Service) Do(ctx context.Context, t time.Time) (err error) {

	switch s.model {
	case TilingModel:
		err = s.Tiling(ctx, t)

	case IterationModel:
		err = s.Iteration(ctx, t)

	default:

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

	if ok != nil {
		s.zl.Sugar().Error(ok)
		return ok
	}

	if s.cfg.MaxRobotGoroutines == 0 {
		ok := s.DoStepAndEvents(ctx, nil)
		if ok != nil {
			s.zl.Sugar().Info(ok)
		}
	} else {
		count, ok := s.DoAsync(ctx)
		if ok != nil {
			s.zl.Sugar().Info(count, ok)
		}
	}

	return ok
}

func (s *Service) DoAsync(ctx context.Context) (int, error) {

	// TODO: это непосредственно шаг лота
	// переписать на текущий шаг маршрута и семафоры
	processing, ok := s.robotRepository.Processing(ctx)

	cursor := -1
	limit := s.cfg.MaxRobotGoroutines - 1

	var lotsByStream [][]map[string]interface{}

	lots := make([]map[string]interface{}, 0)
	for _, item := range processing {

		if cursor == limit {
			cursor = 0
		} else {
			cursor += 1
		}

		if len(lotsByStream) <= cursor {
			lots = make([]map[string]interface{}, 0)
			if len(lotsByStream) > 0 {
				lotsByStream = append(lotsByStream, lots) // TODO: убрать
			}
		}

		lots = append(lots, item)

	}

	if len(lots) > 0 {
		lotsByStream = append(lotsByStream, lots) // TODO: убрать
	}

	var wg sync.WaitGroup
	for _, items := range lotsByStream {
		wg.Add(1)
		go func(data []map[string]interface{}) {
			defer wg.Done()
			err := s.DoStepAndEvents(ctx, data)
			if err != nil {
				s.zl.Sugar().Info(err)
			}
			err = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		}(items)
	}

	wg.Wait()

	return 0, ok
}

func (s *Service) DoStepAndEvents(ctx context.Context, lots []map[string]interface{}) (result error) {

	result = s.DoIncomingEvents(ctx, lots)
	if result != nil {
		return result
	}

	return s.DoNextStep(ctx, lots)
}

func (s *Service) DoIncomingEvents(ctx context.Context, lots []map[string]interface{}) error {

	events, ok := s.robotRepository.FindEventsPerStep(ctx, lots)
	if ok != nil {
		s.zl.Sugar().Error(ok)
		return ok
	}

	for _, event := range events {
		nodeId := event["node_id"].(int64)
		ok := s.RecordToNextStep(ctx, event, nodeId)
		if ok != nil {
			return ok
		}
	}

	return ok
}

func (s *Service) RecordToNextStep(ctx context.Context, data map[string]interface{}, nodeId int64) (ok error) {

	updated, ok := s.robotRepository.UpdateProcessing(ctx, data, nodeId)
	s.zl.Sugar().Info(data, updated)
	if ok != nil {
		s.zl.Sugar().Error(ok)
	}

	return ok
}

func (s *Service) DoNextStep(ctx context.Context, lots []map[string]interface{}) (ok error) {

	for _, lot := range lots {
		ok = s.DoNextStepQuery(ctx, lot)
	}

	return ok
}

func (s *Service) DoNextStepQuery(ctx context.Context, data map[string]interface{}) error {

	t := data["type"]
	var err error

	switch t {
	case action:
		_action := data["action"]
		err = s.DoAction(ctx, _action, data)
		if err == nil {
			err = s.StepToNextNode(ctx, data)
		}
	case wait:
		w := data["waiting_time"].(int32)
		e := data["entry_time"].(time.Time)

		leaveSeconds := time.Now().Second() + int(w)
		timeLine := time.Unix(int64(leaveSeconds), 0)
		if e.Before(timeLine) {
			err = s.StepToNextNode(ctx, data)
		}
	case terminate:
		err = s.Terminate(ctx, data)
	default:
		err = s.StepToNextNode(ctx, data)
	}

	return err
}

func (s *Service) DoAction(ctx context.Context, a interface{}, data map[string]interface{}) error {
	action := a.(string)
	return s.InvokeAction(ctx, action, data)
}

func (s *Service) StepToNextNode(ctx context.Context, data map[string]interface{}) error {

	nextNode, ok := s.FindNextNode(ctx, data["node_id"])
	if ok == nil && nextNode != 0 {
		s.zl.Sugar().Warn("Next step...")
		return s.RecordToNextStep(ctx, data, nextNode)
	}

	return ok
}

func (s *Service) FindNextNode(ctx context.Context, i interface{}) (int64, error) {

	_sql, args, err := squirrel.StatementBuilder.
		Select("*").
		From("_Ref_M as nodes").
		Where(squirrel.Gt{"id": i}).
		OrderBy("id").
		Limit(1).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return 0, err
	}

	results, err := s.robotRepository.RootRepository.Get(ctx, _sql, args...)
	if err != nil {
		return 0, err
	}

	for _, result := range results {
		return result["id"].(int64), nil
	}

	return 0, nil

}

func (s *Service) Terminate(ctx context.Context, data map[string]interface{}) error {

	_sql, args, err := squirrel.
		StatementBuilder.
		Delete("_InfoReg_CSR").
		Where(squirrel.Eq{"id": data["proc_id"]}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	return s.robotRepository.RootRepository.Delete(ctx, _sql, args...)
}

func (s *Service) InvokeAction(ctx context.Context, name string, data map[string]interface{}) error {

	var args []interface{}
	args = append(args, ctx)
	args = append(args, data)

	inputs := make([]reflect.Value, len(args))
	for i := range args {
		inputs[i] = reflect.ValueOf(args[i])
	}

	t := s.action
	values := reflect.ValueOf(t).MethodByName(name).Call(inputs)
	err := values[0].Interface()
	if err != nil {
		return err.(error)
	}

	return nil
}
