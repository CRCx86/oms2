package robot

import (
	"context"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"oms2/internal/pkg/service/log"
	v7 "oms2/internal/pkg/storage/elastic/v7"
	"oms2/internal/pkg/util"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"

	"oms2/internal/oms"
	"oms2/internal/pkg/repository/robot"
)

const (
	IterationModel   = "Iteration"
	TilingModel      = "Tiling"
	MultiTilingModel = "MultiTiling"
)

const (
	action    = "action"
	wait      = "wait"
	terminate = "terminate"
)

const (
	PrefixKeyThreadManager = "ManagerThreadRun"
)

type Service struct {
	zl  *zap.Logger
	cfg *oms.Config

	ticker         *time.Ticker
	restartTimeOut time.Duration

	done  chan bool
	model string
	wg    sync.WaitGroup

	action          *Action
	robotRepository *robot.Repository
	logger          *log.Service

	managers map[string]chan int
	robotCh  chan bool
	running  bool
}

func NewService(cfg *oms.Config, action *Action, r *robot.Repository, logger *log.Service, zl *zap.Logger) *Service {
	return &Service{
		zl:              zl,
		cfg:             cfg,
		ticker:          time.NewTicker(1 * time.Second),
		restartTimeOut:  10 * time.Second,
		done:            make(chan bool),
		model:           TilingModel,
		action:          action,
		robotRepository: r,
		logger:          logger,
		managers:        make(map[string]chan int),
		robotCh:         make(chan bool),
	}
}

func (s *Service) Start(_ context.Context) error {

	c, cancel := context.WithTimeout(context.Background(), s.cfg.MaxCollectTime)

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt)

	go func() {

		defer func() {
			s.logger.LogMessage(c, v7.SystemMessage, "Остановка сервиса...", v7.SystemIndex, util.EmptyDataStruct())
		}()

		s.running = true

		for {
			select {
			case res := <-s.done:
				if res {
					s.ticker.Stop()

					s.logger.LogMessage(c, v7.SystemMessage, "Остановка тикера", v7.SystemIndex, util.EmptyDataStruct())

					cancel()
					return
				}
			case t := <-s.ticker.C:

				err := s.Do(c, t)
				if err != nil {
					s.zl.Sugar().Error(err)
					s.ticker.Stop()

					go func() {
						s.done <- true
					}()
				}

			default:

			}

		}
	}()

	go func() {
		for {
			select {
			case sig := <-signals:
				message := fmt.Sprintf("Got %s signals. Aborting...", sig)
				s.logger.LogMessage(c, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())

				go func() {
					s.robotCh <- true
					s.done <- true
				}()

				return
			}
		}

	}()

	message := fmt.Sprintf("Robot has started")
	s.logger.LogMessage(c, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())

	return nil

}

func (s *Service) Stop(_ context.Context) error {
	go func() {
		s.done <- true
	}()

	return nil
}

func (s *Service) Do(ctx context.Context, t time.Time) (err error) {

	switch s.model {
	case IterationModel:
		err = s.Iteration(ctx, t)
	case TilingModel:
		if !s.running {
			break
		}
		err = s.Tiling(ctx, t)
	case MultiTilingModel:
		if !s.running {
			break
		}
		err = s.MultiTiling(ctx, t)

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

// DoStep Iteration model
func (s *Service) DoStep(ctx context.Context) (ok error) {

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

	lotsOrdersNoGroup, ok := s.robotRepository.GetOrderByLotsFromProcessingRegister(ctx)
	if ok != nil {
		return 0, ok
	}

	paramsManager := make(map[string]interface{}, 0)
	paramsManager["cursor"] = s.cfg.MaxRobotGoroutines

	lotsByStream, count := s.DivideLotsByOrders(lotsOrdersNoGroup, paramsManager)

	var wg sync.WaitGroup
	for _, items := range lotsByStream {

		go func() {
			select {
			case res := <-s.robotCh:
				if res {
					s.running = !res
				}
			}
		}()

		if !s.running {
			break
		}

		wg.Add(1)
		go func(data []map[string]interface{}) {
			defer wg.Done()
			err := s.DoStepAndEvents(ctx, data)
			if err != nil {
				s.zl.Sugar().Info(err)
			}
		}(items)
	}

	wg.Wait()

	return count, ok
}

// Tiling tiling model
func (s *Service) Tiling(ctx context.Context, t time.Time) (ok error) {

	message := fmt.Sprintf("Start Tiling manager: %s", t.String())
	s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())

	startTime := time.Now()
	for {

		go func() {
			select {
			case res := <-s.robotCh:
				if res {
					s.running = !res
				}
			default:
			}
		}()

		isContinue := s.running && (time.Now().Sub(startTime) < s.restartTimeOut)
		if !isContinue {
			break
		}

		paramsManager := make(map[string]interface{}, 0)
		paramsManager["cursor"] = s.cfg.MaxRobotGoroutines
		paramsManager["group"] = -1

		registerActivityList, ok := s.robotRepository.GetRegisterActivityList(ctx)
		if ok != nil {
			return ok
		}

		if len(registerActivityList) < s.cfg.MaxRobotGoroutines && len(s.managers) == 0 {

			if len(registerActivityList) > 0 {
				paramsManager["cursor"] = s.cfg.MaxRobotGoroutines - len(registerActivityList) - 1
			}

			//uid := uuid.NewV4().String()
			uid := PrefixKeyThreadManager
			manager := make(chan int) // канал для менеджера потоков

			s.managers[uid] = manager
			go s.TilingThreadManager(ctx, uid, manager, paramsManager)

			message := fmt.Sprintf("running managers: %d", len(s.managers))
			s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())
		}

		for _, data := range s.managers {
			go func(ch chan int) {
				select {
				case item := <-ch:
					message := fmt.Sprintf("manager data: %d", item)
					s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())
				default:
				}
			}(data)
		}

		time.Sleep(100 * time.Millisecond)
	}

	message = fmt.Sprintf("End Tiling manager: %d", time.Now().Sub(startTime))
	s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())

	return ok
}

func (s *Service) MultiTiling(ctx context.Context, t time.Time) (ok error) {

	message := fmt.Sprintf("Start MultiTilingModel manager: %s", t.String())
	s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())

	startTime := time.Now()
	for {

		go func() {
			select {
			case res := <-s.robotCh:
				if res {
					s.running = !res
				}
			default:
			}
		}()

		isContinue := s.running && (time.Now().Sub(startTime) < s.restartTimeOut)
		if !isContinue {
			break
		}

		paramsManager := make(map[string]interface{}, 0)
		paramsManager["cursor"] = s.cfg.MaxRobotGoroutines

		registerActivityList, ok := s.robotRepository.GetRegisterActivityList(ctx)
		if ok != nil {
			return ok
		}

		groupList, ok := s.robotRepository.ProcessingGroupList(ctx)
		if ok != nil {
			return ok
		}

		for _, val := range groupList {

			gpId := val["group_id"].(int32)
			threadKey := PrefixKeyThreadManager + strconv.Itoa(int(gpId))

			activityCount := 0
			if s.managers[threadKey] == nil {
				for _, v := range registerActivityList {
					groupId := v["group_id"].(int32)
					if gpId == groupId {
						activityCount += 1
					}
				}

				if s.cfg.MaxRobotGoroutines >= activityCount {

					paramsManager["cursor"] = s.cfg.MaxRobotGoroutines - activityCount
					paramsManager["group"] = gpId

					manager := make(chan int) // канал для менеджера потоков
					s.managers[threadKey] = manager

					go s.TilingThreadManager(ctx, threadKey, manager, paramsManager)

					message := fmt.Sprintf("running managers: %d", len(s.managers))
					s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())
				}
			}
		}

		if len(registerActivityList) < s.cfg.MaxRobotGoroutines && len(s.managers) == 0 {

		}

		for _, data := range s.managers {
			go func(ch chan int) {
				select {
				case item := <-ch:
					message := fmt.Sprintf("manager data: %d", item)
					s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())
				default:
				}
			}(data)
		}

		time.Sleep(100 * time.Millisecond)
	}

	message = fmt.Sprintf("End Tiling manager: %d", time.Now().Sub(startTime))
	s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())

	return ok
}

func (s *Service) TilingThreadManager(ctx context.Context, uid string, manager chan int, params map[string]interface{}) {

	defer func() {
		delete(s.managers, uid)
		defer close(manager)
	}()

	lotsOrdersNoGroup, ok := s.robotRepository.GetOrderByLotsFromProcessingRegisterAndRegisterActivity(ctx, params)
	if ok != nil {
		s.zl.Sugar().Info(ok)
		return
	}
	lotsByStream, count := s.DivideLotsByOrders(lotsOrdersNoGroup, params)
	for _, items := range lotsByStream {

		uid := uuid.NewV4().String()

		activity := make(map[int32]map[string]interface{}, 0)
		for _, item := range items {
			order := item["order_id"].(int32)
			if activity[order] == nil {
				itemMap := make(map[string]interface{})
				itemMap["order_id"] = item["order_id"]
				itemMap["thread_key"] = uid
				itemMap["start_time"] = time.Now()
				itemMap["thread_id"] = uid
				itemMap["group_id"] = params["group"]
				activity[order] = itemMap
			}
		}

		activityData := make([]map[string]interface{}, 0)
		for _, v := range activity {
			activityData = append(activityData, v)
		}
		_, ok := s.robotRepository.UpdateProcessingActivity(ctx, activityData, "")
		if ok != nil {
			s.zl.Sugar().Info(ok)
			return
		}

		go func(data []map[string]interface{}, uid string) {
			err := s.Shard_DoStepAndEvents(ctx, data, uid)
			if err != nil {
				s.zl.Sugar().Info(err)
			}
		}(items, uid)
	}

	manager <- count

}

func (s *Service) Shard_DoStepAndEvents(ctx context.Context, data []map[string]interface{}, uid string) (result error) {

	result = s.DoStepAndEvents(ctx, data)
	if result != nil {
		return result
	}

	result = s.robotRepository.DeleteFromRegisterActivityByThreadId(ctx, uid)

	return result
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

	_, ok = s.robotRepository.UpdateProcessing(ctx, data, nodeId)
	//s.zl.Sugar().Info(data, updated)
	if ok != nil {
		s.zl.Sugar().Error(ok)
	}

	return ok
}

func (s *Service) DoNextStep(ctx context.Context, lots []map[string]interface{}) (ok error) {

	results, ok := s.robotRepository.Processing(ctx, lots)
	if ok != nil {
		return ok
	}

	for _, lot := range results {
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
		if err == nil {
			message := fmt.Sprintf("Terminate: lot - %d, proc - %d", data["lot_id"], data["proc_id"])
			s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())
		}
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
		message := fmt.Sprintf("Next step...")
		s.logger.LogMessage(ctx, v7.SystemMessage, message, v7.SystemIndex, util.EmptyDataStruct())
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

func (s *Service) DivideLotsByOrders(lotsOrdersNoGroup []map[string]interface{}, params map[string]interface{}) ([][]map[string]interface{}, int) {

	lotsByStream := make([][]map[string]interface{}, 0)

	lotsOrderGroup := make(map[interface{}][]map[string]interface{})
	for _, item := range lotsOrdersNoGroup {
		lotsOrderGroup[item["order_id"]] = append(lotsOrderGroup[item["order_id"]], item)
	}

	cursor := -1
	limit := params["cursor"].(int) - 1

	count := 0
	for _, value := range lotsOrderGroup {

		if cursor == limit {
			cursor = 0
		} else {
			cursor += 1
		}

		if len(lotsByStream) <= cursor {
			lotsByStream = append(lotsByStream, make([]map[string]interface{}, 0))
		}

		for _, item := range value {
			if item["thread"].(int32) >= 900 {

			} else {
				lotsByStream[cursor] = append(lotsByStream[cursor], item)
			}
			count += 1
		}

	}
	return lotsByStream, count
}
