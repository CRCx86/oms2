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
