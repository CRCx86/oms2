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
	events, err := robotRepo.FindEventsPerStep(ctx)
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
	events, err := robotRepo.FindEventsPerStep(ctx)
	require.NoError(t, err)
	require.Greater(t, len(events), 1)

	updated, ok := robotRepo.UpdateProcessing(ctx, events[0])
	require.NoError(t, ok)
	require.Greater(t, updated, uint(0))
}

func PrepareTestDB(ctx context.Context, conn *pgxpool.Pool) error {

	qs := []string{
		`DROP TABLE IF EXISTS event_semaphores;`,
		`DROP TABLE IF EXISTS nodes_event_types;`,
		`DROP TABLE IF EXISTS events;`,
		`DROP TABLE IF EXISTS event_types;`,
		`DROP TABLE IF EXISTS lots_nodes;`,
		`DROP TABLE IF EXISTS lots;`,
		`DROP TABLE IF EXISTS nodes;`,
		`DROP TABLE IF EXISTS orders;`,
		`DROP TABLE IF EXISTS shipments;`,
		`DROP TABLE IF EXISTS deliveries;`,
		`DROP TABLE IF EXISTS users;`,

		`CREATE TABLE event_types (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO event_types(name) 
			VALUES('event_type1'), ('event_type2');`,

		`CREATE TABLE lots (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO lots(name) 
			VALUES ('lot1'), ('lot2');`,

		`CREATE TABLE events (
    		id bigserial primary key,
    		name varchar NOT NULL,
			event_type_id int,
			lot_id int,
			foreign key(event_type_id) references event_types(id) on delete cascade,
			foreign key(lot_id) references lots(id) on delete cascade);`,
		`INSERT INTO events(name, event_type_id, lot_id) 
			VALUES('event1', 1, 1), ('event1', 2, 2);`,

		`CREATE TABLE nodes (
    		id bigserial primary key,
    		name varchar NOT NULL,
			type varchar NOT NULL,
			action varchar NOT NULL,
			waiting_time int);`,
		`INSERT INTO nodes(name, type, action, waiting_time) 
			VALUES ('node1', 'action', 'FirstInit', 0), 
			('node2', 'action', 'SecondInit', 0),
			('node3', 'wait', 'Wait', 120), 
			('node4', 'trigger', 'Trigger', 0),
			('node5', 'terminate', 'Terminate', 0);`,

		`CREATE TABLE orders (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO orders(name) 
			VALUES('order1'), ('order2');`,

		`CREATE TABLE shipments (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO shipments(name) 
			VALUES('shipment1'), ('shipment1');`,

		`CREATE TABLE deliveries (
    		id bigserial primary key,
    		name varchar NOT NULL);`,
		`INSERT INTO deliveries(name) 
			VALUES('delivery1'), ('delivery1');`,

		`CREATE TABLE users (
    		id bigserial primary key,
    		name varchar NOT NULL,
			age integer);`,
		`INSERT INTO users(name, age) 
			VALUES('user1', 1), ('user2', 2);`,

		// таблица процессинга (теукщий шаг)
		`CREATE TABLE lots_nodes (
		  id bigserial,
		  lot_id    int REFERENCES lots (id) ON UPDATE CASCADE ON DELETE CASCADE, 
          node_id 	int REFERENCES nodes (id) ON UPDATE CASCADE,
		  thread	int NOT NULL DEFAULT 1,
		  entry_time timestamp WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
		  CONSTRAINT lots_nodes_pkey PRIMARY KEY (lot_id, node_id)
		);`,
		`INSERT INTO lots_nodes(lot_id, node_id) 
			VALUES(1, 1), (2, 1);`,

		// табличная часть узла
		`CREATE TABLE nodes_event_types (
		  id 			bigserial,
		  node_id    	int REFERENCES nodes (id) ON UPDATE CASCADE ON DELETE CASCADE, 
          event_type_id int REFERENCES event_types (id) ON UPDATE CASCADE,
		  CONSTRAINT nodes_event_types_pkey PRIMARY KEY (node_id, event_type_id)
		);`,
		`INSERT INTO nodes_event_types(node_id, event_type_id) 
			VALUES(3, 1), (3, 2), (4, 1);`,

		// семафоры обработки событий, техн.
		`CREATE TABLE event_semaphores (
		  id bigserial,
		  lot_id    int REFERENCES lots (id) ON UPDATE CASCADE ON DELETE CASCADE, 
          semaphore_id int REFERENCES event_types (id) ON UPDATE CASCADE,
		  event_id  int REFERENCES events (id),
		  CONSTRAINT event_semaphores_pkey PRIMARY KEY (lot_id, semaphore_id)
		);`,
		`INSERT INTO event_semaphores(lot_id, semaphore_id, event_id) 
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
