package robot

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"go.uber.org/zap"

	"oms2/internal/pkg/repository/root"
	"oms2/internal/pkg/storage/postgres"
)

//var (
//	ErrBadModel         = errors.New("bad model")
//	ErrValidationFailed = errors.New("validation failed")
//)

type Repository struct {
	zl             *zap.Logger
	storage        *postgres.Postgres
	RootRepository *root.Repository
}

func NewRepository(s *postgres.Postgres, root *root.Repository, zl *zap.Logger) *Repository {
	return &Repository{
		zl:             zl,
		storage:        s,
		RootRepository: root,
	}
}

func (r *Repository) Processing(ctx context.Context, lots []map[string]interface{}) ([]map[string]interface{}, error) {

	lotsId := make([]int32, 0)
	for _, lot := range lots {
		lotsId = append(lotsId, lot["lot_id"].(int32))
	}

	_sql, args, err := squirrel.StatementBuilder.
		Select("ln.thread as thread," +
			"ln.id as proc_id," +
			"l.id as lot_id," +
			"n.id as node_id," +
			"n.action as action," +
			"n.name as name," +
			"n.type as type," +
			"n.waiting_time as waiting_time," +
			"ln.entry_time as entry_time").
		From("_InfoReg_CSR as ln").
		InnerJoin("_Ref_L as l ON ln.lot_id = l.id").
		InnerJoin("_Ref_M as n ON ln.node_id = n.id").
		Where(squirrel.Eq{"lot_id": lotsId}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		r.zl.Sugar().Error(err)
		return nil, err
	}

	return r.RootRepository.Get(ctx, _sql, args...)
}

func (r *Repository) FindEventsPerStep(ctx context.Context, lots []map[string]interface{}) ([]map[string]interface{}, error) {

	nodes, _, err := squirrel.Select("nodes.id as node_id," +
		"nodes.type as node_type," +
		"net.event_type_id as event_type_id," +
		"nodes.event_trigger as event_trigger").
		From("_Ref_M as nodes").
		LeftJoin("_RefVT_ME as net on net.node_id = nodes.id").
		Where("case when nodes.type = 'action' and net.event_type_id is null then false else true end").
		ToSql()
	if err != nil {
		r.zl.Sugar().Error(err)
		return nil, err
	}

	if lots != nil {

		var lotsId []interface{}
		for _, lot := range lots {
			lotsId = append(lotsId, lot["lot_id"])
		}

		_sql, args, err := squirrel.StatementBuilder.
			Select(
				"ltnds.id as proc_id," +
					"events.lot_id as lot_id," +
					"events.event_type_id as event_type_id," +
					"ne.node_id as node_id," +
					"ltnds.node_id as prev_id").
			From("_InfoReg_ES as semaphores").
			LeftJoin("_Ref_E as events on semaphores.event_id = events.id").
			InnerJoin("_InfoReg_CSR as ltnds on events.lot_id = ltnds.lot_id").
			InnerJoin(fmt.Sprintf("(%s) as nodes on events.event_type_id = nodes.event_type_id "+
				"and ltnds.node_id = nodes.node_id", nodes)).
			InnerJoin(fmt.Sprintf("(%s) as ne on events.event_type_id = ne.event_trigger "+
				"and nodes.node_id <= ne.node_id", nodes)).
			Where(squirrel.Eq{"semaphores.lot_id": lotsId}).
			PlaceholderFormat(squirrel.Dollar).
			ToSql()
		if err != nil {
			r.zl.Sugar().Error(err)
			return nil, err
		}

		return r.RootRepository.Get(ctx, _sql, args...)

	}

	_sql, args, err := squirrel.StatementBuilder.
		Select(
			"ltnds.id as proc_id," +
				"events.lot_id as lot_id," +
				"events.event_type_id as event_type_id," +
				"ne.node_id as node_id," +
				"ltnds.node_id as prev_id").
		From("_InfoReg_ES as semaphores").
		LeftJoin("_Ref_E as events on semaphores.event_id = events.id").
		InnerJoin("_InfoReg_CSR as ltnds on events.lot_id = ltnds.lot_id").
		InnerJoin(fmt.Sprintf("(%s) as nodes on events.event_type_id = nodes.event_type_id "+
			"and ltnds.node_id = nodes.node_id", nodes)).
		InnerJoin(fmt.Sprintf("(%s) as ne on events.event_type_id = ne.event_trigger "+
			"and nodes.node_id <= ne.node_id", nodes)).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()
	if err != nil {
		r.zl.Sugar().Error(err)
		return nil, err
	}

	return r.RootRepository.Get(ctx, _sql, args...)

}

func (r *Repository) UpdateProcessing(ctx context.Context, data map[string]interface{}, nodeId int64) (uint, error) {

	var _sql string
	var args []interface{}
	var err error

	if data["proc_id"] == 0 {

		_sql, args, err = squirrel.
			StatementBuilder.
			PlaceholderFormat(squirrel.Dollar).
			Insert("_InfoReg_CSR").
			Columns("lot_id", "node_id", "entry_time").
			Values(data["lotId"], nodeId, time.Now()).
			Suffix("RETURNING id").
			ToSql()

		if err != nil {
			r.zl.Sugar().Error(err)
			return 0, err
		}

	} else {

		values := make(map[string]interface{})
		values["node_id"] = nodeId
		values["entry_time"] = time.Now()

		_sql, args, err = squirrel.
			StatementBuilder.
			PlaceholderFormat(squirrel.Dollar).
			Update("_InfoReg_CSR").
			SetMap(values).
			Where(squirrel.Eq{"id": data["proc_id"]}).
			Suffix("RETURNING id").
			ToSql()

		if err != nil {
			r.zl.Sugar().Error(err)
			return 0, err
		}

	}

	return r.RootRepository.CreateOrUpdate(ctx, _sql, args...)
}

func (r *Repository) UpdateProcessingActivity(ctx context.Context, data []map[string]interface{}, threadKey string) (uint, error) {

	var _sql string
	var args []interface{}
	var err error

	if len(threadKey) == 0 {

		insertQuery := squirrel.
			StatementBuilder.
			PlaceholderFormat(squirrel.Dollar).
			Insert("_InfoReg_PA")

		for _, item := range data {
			insertQuery = insertQuery.
				SetMap(item)
		}

		_sql, args, err = insertQuery.Suffix("RETURNING id").ToSql()

		if err != nil {
			r.zl.Sugar().Error(err)
			return 0, err
		}

	} else {

		updateQuery := squirrel.
			StatementBuilder.
			PlaceholderFormat(squirrel.Dollar).
			Update("_InfoReg_PA")

		for _, item := range data {
			updateQuery = updateQuery.
				SetMap(item)
		}

		_sql, args, err = updateQuery.
			Where(squirrel.Eq{"id": threadKey}).
			Suffix("RETURNING id").
			ToSql()

		if err != nil {
			r.zl.Sugar().Error(err)
			return 0, err
		}

	}

	return r.RootRepository.CreateOrUpdate(ctx, _sql, args...)
}

func (r *Repository) GetOrderByLotsFromProcessingRegister(ctx context.Context) ([]map[string]interface{}, error) {

	_sql := `select
				inner_query.lot_id as lot_id,
				lots.order_id as order_id,
				inner_query.thread as thread,
				sum(inner_query.weight) as weight
			from (select
				   csr.lot_id as lot_id,
				   csr.weight as weight,
				   case when csr.thread >= 900 then csr.thread else 0 end as thread
			from _inforeg_csr as csr
			where csr.next_run_time <= $1
			union
			select
				   es.lot_id,
				   max(5000),
				   max(0)
			from _inforeg_es as es
				inner join _inforeg_csr ic on es.lot_id = ic.lot_id
				inner join _refvt_me rme on ic.node_id = rme.node_id
					and rme.event_type_id = es.semaphore_id
			where es.entry_time >= $2
			group by
				es.lot_id) as inner_query
			
			left join _ref_l as lots on inner_query.lot_id = lots.id
			group by
				inner_query.lot_id,
				lots.order_id,
				inner_query.thread
			order by weight desc`

	var args []interface{}
	args = append(args, time.Now())
	args = append(args, time.Now().Add(-24*time.Hour))

	return r.RootRepository.Get(ctx, _sql, args...)
}

func (r *Repository) GetRegisterActivityList(ctx context.Context) ([]map[string]interface{}, error) {

	_sql, args, err := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select("pa.thread_key, pa.thread_id, pa.group_id, max(pa.start_time) as start_time").
		From("_InfoReg_PA as pa").
		GroupBy("thread_key, thread_id, group_id, start_time").
		ToSql()
	if err != nil {
		return nil, err
	}

	return r.RootRepository.Get(ctx, _sql, args...)

}

func (r *Repository) GetOrderByLotsFromProcessingRegisterAndRegisterActivity(ctx context.Context, params map[string]interface{}) ([]map[string]interface{}, error) {

	_sql := ``
	var args []interface{}
	args = append(args, time.Now())
	args = append(args, time.Now().Add(-24*time.Hour))

	groupId := params["group"]
	if groupId == -1 {
		_sql = `select
				inner_query.lot_id as lot_id,
				lots.order_id as order_id,
				inner_query.thread as thread,
				sum(inner_query.weight) as weight
			from (select
				   csr.lot_id as lot_id,
				   csr.weight as weight,
				   case when csr.thread >= 900 then csr.thread else 0 end as thread
			from _inforeg_csr as csr
			where csr.next_run_time <= $1

			union

			select
				   es.lot_id,
				   max(5000),
				   max(0)
			from _inforeg_es as es
				inner join _inforeg_csr ic on es.lot_id = ic.lot_id
				inner join _refvt_me rme on ic.node_id = rme.node_id
					and rme.event_type_id = es.semaphore_id
			where es.entry_time >= $2
			group by
				es.lot_id) as inner_query
			
			left join _ref_l as lots on inner_query.lot_id = lots.id
			left join _inforeg_pa as pa on lots.order_id = pa.order_id

			where pa.order_id is null

			group by
				inner_query.lot_id,
				lots.order_id,
				inner_query.thread
			order by weight desc`
	} else {

		_sql = `select
					inner_query.lot_id as lot_id,
					inner_query.order_id as order_id,
					inner_query.thread as thread,
					sum(inner_query.weight) as weight
				from (select
						  csr.lot_id as lot_id,
						  csr.weight as weight,
						  case when csr.thread >= 900 then csr.thread else 0 end as thread,
						  pg.order_id as order_id
					  from _inforeg_pg as pg
								inner join _ref_o ro on ro.id = pg.order_id
								inner join _ref_l rl on ro.id = rl.order_id
							   inner join _inforeg_csr as csr on rl.id = csr.lot_id
							   left join _inforeg_pa as pa on pg.order_id = pa.order_id
					  where
							  csr.next_run_time <= $1
						and pa.order_id is null
						and pg.group_id = $3
				
					  union all
				
					  select
						  es.lot_id,
						  max(5000),
						  max(0),
						  pg.order_id as order_id
					  from _inforeg_pg as pg
							   inner join _inforeg_es as es on pg.order_id = es.order_id
							   inner join _inforeg_csr csr on es.lot_id = csr.lot_id
							   inner join _refvt_me rme on csr.node_id = rme.node_id
						  and rme.event_type_id = es.semaphore_id
							   left join _inforeg_pa as pa
										 on pg.order_id = pa.order_id
					  where
							  es.entry_time >= $2
						and csr.next_run_time > $1
						and pa.order_id is null
						and pg.group_id = $3
					  group by
						  es.lot_id,
						  pg.order_id
					 ) as inner_query
				
				group by
					inner_query.lot_id,
					inner_query.order_id,
					inner_query.thread
				order by weight desc`

		args = append(args, groupId)
	}

	return r.RootRepository.Get(ctx, _sql, args...)
}

func (r *Repository) DeleteFromRegisterActivityByThreadId(ctx context.Context, uid string) error {

	_sql, args, err := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Delete("_InfoReg_PA").
		Where(squirrel.Eq{"thread_key": uid}).
		ToSql()

	if err != nil {
		return err
	}

	return r.RootRepository.Delete(ctx, _sql, args...)

}

func (r *Repository) ProcessingGroupList(ctx context.Context) ([]map[string]interface{}, error) {

	_sql, args, err := squirrel.
		StatementBuilder.
		PlaceholderFormat(squirrel.Dollar).
		Select("pg.group_id as group_id").
		From("_InfoReg_PG as pg").
		GroupBy("group_id").
		ToSql()
	if err != nil {
		return nil, err
	}

	return r.RootRepository.Get(ctx, _sql, args...)
}
