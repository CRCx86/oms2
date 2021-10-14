package robot

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"oms2/internal/pkg/repository/root"
	postgres2 "oms2/internal/pkg/storage/postgres"
	"testing"
	"time"
)

func TestRepository_PrepareTestDB(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	zl := zap.L()
	p := postgres2.NewPostgres(postgres2.Config{
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "oms",
		LogLevel:   "error",
	}, zl)
	err := p.Start(ctx)
	require.NoError(t, err)
	defer p.Stop(ctx)

	conn, err := p.Conn(ctx)
	require.NoError(t, err)
	err = PrepareTestDB(ctx, conn)
	require.NoError(t, err)
}

func TestRepository_Processing(t *testing.T) {

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	zl := zap.L()
	p := postgres2.NewPostgres(postgres2.Config{
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "oms",
		LogLevel:   "error",
	}, zl)
	err := p.Start(ctx)
	require.NoError(t, err)
	defer p.Stop(ctx)

	conn, err := p.Conn(ctx)
	require.NoError(t, err)

	err = PrepareTestDB(ctx, conn)
	require.NoError(t, err)

	rootRepo := root.NewRepository(p, zl)

	robotRepo := NewRepository(p, rootRepo, zl)
	processing, err := robotRepo.Processing(ctx)
	require.NoError(t, err)
	require.Greater(t, len(processing), 1)
}

func TestRepository_FindEventsPerStep(t *testing.T) {

	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	zl := zap.L()
	p := postgres2.NewPostgres(postgres2.Config{
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "oms",
		LogLevel:   "error",
	}, zl)
	err := p.Start(ctx)
	require.NoError(t, err)
	defer p.Stop(ctx)

	conn, err := p.Conn(ctx)
	require.NoError(t, err)

	err = PrepareTestDB(ctx, conn)
	require.NoError(t, err)

	rootRepo := root.NewRepository(p, zl)

	robotRepo := NewRepository(p, rootRepo, zl)
	events, err := robotRepo.FindEventsPerStep(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, len(events), 1)
}

func TestRepository_RecordToNextStep(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	zl := zap.L()
	p := postgres2.NewPostgres(postgres2.Config{
		DBUser:     "postgres",
		DBPassword: "postgres",
		DBHost:     "localhost",
		DBPort:     "5432",
		DBName:     "oms",
		LogLevel:   "error",
	}, zl)
	err := p.Start(ctx)
	require.NoError(t, err)
	defer p.Stop(ctx)

	conn, err := p.Conn(ctx)
	require.NoError(t, err)

	err = PrepareTestDB(ctx, conn)
	require.NoError(t, err)

	rootRepo := root.NewRepository(p, zl)

	robotRepo := NewRepository(p, rootRepo, zl)
	events, err := robotRepo.FindEventsPerStep(ctx, nil)
	require.NoError(t, err)
	require.Greater(t, len(events), 1)

	updated, ok := robotRepo.UpdateProcessing(ctx, events[0], 0)
	require.NoError(t, ok)
	require.Greater(t, updated, uint(0))
}

func PrepareTestDB(ctx context.Context, conn *pgxpool.Pool) error {

	qs := []string{
		`DROP TABLE IF EXISTS _InfoReg_ES;`,
		`DROP TABLE IF EXISTS _RefVT_ME;`,
		`DROP TABLE IF EXISTS _Ref_E;`,
		`DROP TABLE IF EXISTS _Ref_ET;`,
		`DROP TABLE IF EXISTS _InfoReg_CSR;`,
		`DROP TABLE IF EXISTS _Ref_L;`,
		`DROP TABLE IF EXISTS _Ref_M;`,
		`DROP TABLE IF EXISTS _Ref_O;`,
		`DROP TABLE IF EXISTS _Ref_S;`,
		`DROP TABLE IF EXISTS _Ref_D;`,

		`CREATE TABLE _Ref_ET (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO _Ref_ET(name) 
			VALUES('event_type1'), ('event_type2');`,

		`CREATE TABLE _Ref_L (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO _Ref_L(name) 
			VALUES ('lot1'), ('lot2');`,

		`CREATE TABLE _Ref_E (
    		id bigserial primary key,
    		name varchar NOT NULL,
			event_type_id int,
			lot_id int,
			foreign key(event_type_id) references _Ref_ET(id) on delete cascade,
			foreign key(lot_id) references _Ref_L(id) on delete cascade);`,
		`INSERT INTO _Ref_E(name, event_type_id, lot_id) 
			VALUES('event1', 1, 1), ('event2', 2, 2);`,

		`CREATE TABLE _Ref_M (
    		id bigserial primary key,
    		name varchar NOT NULL,
			type varchar NOT NULL,
			action varchar NOT NULL,
			event_trigger int,
			waiting_time int);`,
		`INSERT INTO _Ref_M(name, type, action, event_trigger, waiting_time) 
			VALUES ('node1', 'action', 'FirstInit', null, 0), 
			('node2', 'action', 'SecondInit', null, 0),
			('node3', 'wait', 'Wait', null, 120), 
			('node4', 'trigger', 'Trigger', 1, 0),
			('node5', 'terminate', 'Terminate', null, 0);`,

		`CREATE TABLE _Ref_O (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO _Ref_O(name) 
			VALUES('order1'), ('order2');`,

		`CREATE TABLE _Ref_S (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO _Ref_S(name) 
			VALUES('shipment1'), ('shipment1');`,

		`CREATE TABLE _Ref_D (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO _Ref_D(name) 
			VALUES('delivery1'), ('delivery1');`,

		// таблица процессинга (теукщий шаг)
		`CREATE TABLE _InfoReg_CSR (
		  id bigserial,
		  lot_id    int REFERENCES _Ref_L (id) ON UPDATE CASCADE ON DELETE CASCADE, 
          node_id 	int REFERENCES _Ref_M (id) ON UPDATE CASCADE,
		  thread	int NOT NULL DEFAULT 1,
		  entry_time timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		  CONSTRAINT _InfoReg_CSR_pkey PRIMARY KEY (lot_id, node_id)
		);`,
		`INSERT INTO _InfoReg_CSR(lot_id, node_id) 
			VALUES(1, 1), (2, 1);`,

		// табличная часть узла
		`CREATE TABLE _RefVT_ME (
		  id 			bigserial,
		  node_id    	int REFERENCES _Ref_M (id) ON UPDATE CASCADE ON DELETE CASCADE, 
          event_type_id int REFERENCES _Ref_ET (id) ON UPDATE CASCADE,
		  CONSTRAINT _RefVT_ME_pkey PRIMARY KEY (node_id, event_type_id)
		);`,
		`INSERT INTO _RefVT_ME(node_id, event_type_id) 
			VALUES(3, 1), (3, 2), (4, 1);`,

		// семафоры обработки событий, техн.
		`CREATE TABLE _InfoReg_ES (
		  id bigserial,
		  lot_id    int REFERENCES _Ref_L (id) ON UPDATE CASCADE ON DELETE CASCADE, 
          semaphore_id int REFERENCES _Ref_ET (id) ON UPDATE CASCADE,
		  event_id  int REFERENCES _Ref_E (id),
		  CONSTRAINT _InfoReg_ES_pkey PRIMARY KEY (lot_id, semaphore_id)
		);`,
		`INSERT INTO _InfoReg_ES(lot_id, semaphore_id, event_id) 
			VALUES(1, 1, 1), (2, 2, 2);`,
	}

	for _, q := range qs {
		_, err := conn.Exec(ctx, q)
		if err != nil {
			return err
		}
	}

	return nil
}
