package v7

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	elasticReadyHealthStatus = "green"
	pageSize                 = 1000
	keepAliveCursorTime      = "5m"
)

type Config struct {
	Url                 string `envconfig:"url" default:"http://localhost:9200"`
	Login               string `envconfig:"login"`
	Password            string `envconfig:"password"`
	Sniff               bool   `envconfig:"sniff" default:"false"`
	HealthCheckInterval int    `envconfig:"health_check_interval" default:"10"`
}

func NewElastic(cfg Config, zl *zap.Logger) *Elastic {
	return &Elastic{
		cfg:    cfg,
		logger: zl,
	}
}

type Elastic struct {
	cfg    Config
	client *elastic.Client
	health *elastic.ClusterHealthService
	logger *zap.Logger
}

func (e *Elastic) Client() *elastic.Client {
	return e.client
}

func (e *Elastic) Start(ctx context.Context) error {
	client, err := elastic.NewClient(
		elastic.SetURL(e.cfg.Url),
		elastic.SetSniff(e.cfg.Sniff),
		elastic.SetHealthcheck(true),
		elastic.SetHealthcheckInterval(time.Duration(e.cfg.HealthCheckInterval)*time.Second),
		elastic.SetBasicAuth(e.cfg.Login, e.cfg.Password),
		elastic.SetErrorLog(NewElasticErrorLogger(e.logger)),
	)
	if err != nil {
		return err
	}

	_, err = client.NodesInfo().Do(ctx)
	if err != nil {
		return err
	}

	e.client = client
	e.health = client.ClusterHealth()

	return nil
}

func (e *Elastic) Stop(_ context.Context) error {
	e.client.Stop()
	return nil
}

func (e *Elastic) IsReady(ctx context.Context) bool {
	resp, err := e.health.Do(ctx)
	if err != nil {
		e.logger.Error("cannot check elastic v7 health", zap.Error(err))
		return false
	}

	if resp.Status != elasticReadyHealthStatus {
		return false
	}

	return true
}

func (e *Elastic) ScrollWithSource(ctx context.Context, indexName string, searchSource *elastic.SearchSource) ([]json.RawMessage, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = opentracing.GlobalTracer().StartSpan("elastic7.Scroll", opentracing.ChildOf(span.Context()))
		defer span.Finish()
	}

	scroll := e.client.Scroll(indexName).
		Scroll(keepAliveCursorTime).
		SearchSource(searchSource).
		Size(pageSize)
	result := make([]json.RawMessage, 0)
	for {
		results, err := scroll.Do(ctx)
		if err == io.EOF {
			_ = scroll.Clear(ctx)
			return result, nil
		}
		if err != nil {
			_ = scroll.Clear(ctx)
			queryText := e.getQuerySource(searchSource)
			err = errors.Wrap(err, fmt.Sprintf("error index: %s with query: %s", indexName, queryText))
			return nil, err
		}

		for _, hit := range results.Hits.Hits {
			select {
			case <-ctx.Done():
				_ = scroll.Clear(context.Background())
				return nil, ctx.Err()
			default:
			}

			result = append(result, hit.Source)
		}
	}
}

func (e *Elastic) getQuerySource(query elastic.Query) string {
	textQuery, _ := query.Source()
	queryText, _ := json.Marshal(textQuery)
	return string(queryText)
}

func NewElasticErrorLogger(l *zap.Logger) *elasticErrorLogger {
	return &elasticErrorLogger{l: l}
}

type elasticErrorLogger struct {
	l *zap.Logger
}

func (e elasticErrorLogger) Printf(format string, v ...interface{}) {
	e.l.Error(fmt.Sprintf(format, v...))
}
