package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/log/zapadapter"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/zap"
)

type Postgres struct {
	conn   *pgxpool.Pool
	conf   Config
	log    *zap.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

type Config struct {
	DBUser     string `envconfig:"db_user" default:"postgres"`
	DBPassword string `envconfig:"db_password" default:"postgres"`
	DBHost     string `envconfig:"db_host" default:"localhost"`
	DBPort     string `envconfig:"db_port" default:"5432"`
	DBName     string `envconfig:"db_name" default:"oms"`
	LogLevel   string `envconfig:"log_level" default:"error"`
}

func NewPostgres(conf Config, log *zap.Logger) *Postgres {
	return &Postgres{conf: conf, log: log}
}

func (p *Postgres) Conn(ctx context.Context) (*pgxpool.Pool, error) {
	ready, err := p.IsReady(ctx)
	if !ready {
		return nil, err
	}

	return p.conn, nil
}

func (p *Postgres) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	conn, err := p.Conn(ctx)
	if err != nil {
		return err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err = tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			p.log.Error("Error with transaction rollback", zap.Error(err))
		}
	}()

	err = fn(ctx, tx)
	if err != nil {
		if err = tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			p.log.Error("Error with transaction rollback", zap.Error(err))
		}

		return err
	}

	return tx.Commit(ctx)
}

func (p *Postgres) Start(ctx context.Context) error {

	dslArray := []string{
		"user=" + p.conf.DBUser,
		"password=" + p.conf.DBPassword,
		"host=" + p.conf.DBHost,
		"port=" + p.conf.DBPort,
		"dbname=" + p.conf.DBName,
		"sslmode=disable",
	}
	dsl := strings.Join(dslArray, " ")

	poolConf, err := pgxpool.ParseConfig(dsl)
	if err != nil {
		return err
	}

	poolConf.HealthCheckPeriod = 1 * time.Second
	poolConf.MaxConns = 10
	poolConf.MinConns = 4
	poolConf.ConnConfig.Logger = zapadapter.NewLogger(p.log)
	logLevel, err := pgx.LogLevelFromString(p.conf.LogLevel)
	if err != nil {
		return err
	}
	poolConf.ConnConfig.LogLevel = logLevel
	poolConf.ConnConfig.PreferSimpleProtocol = true

	p.conn, err = pgxpool.ConnectConfig(ctx, poolConf)
	if err != nil {
		p.log.Error("cannot connect to postgres", zap.Error(err))
		return err
	}

	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.backgroundPing()
	return nil
}

func (p *Postgres) Stop(context.Context) error {
	p.cancel()
	p.conn.Close()
	return nil
}

func (p *Postgres) IsReady(ctx context.Context) (bool, error) {
	if _, err := p.conn.Exec(ctx, ";"); err != nil {
		return false, err
	}

	return true, nil
}

func (p *Postgres) backgroundPing() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-p.ctx.Done():
				break
			case <-ticker.C:
				ready, err := p.IsReady(p.ctx)
				if !ready {
					p.log.Error("ping postgres error", zap.Error(err))
				}
			}
		}
	}()
}
