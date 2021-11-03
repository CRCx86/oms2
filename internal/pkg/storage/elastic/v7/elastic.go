package v7

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"oms2/internal/pkg/config"
	"time"

	"github.com/olivere/elastic/v7"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"oms2/internal/oms"
)

const (
	elasticReadyHealthStatus = "green"
	pageSize                 = 1000
	keepAliveCursorTime      = "5m"

	ErrorIndex = "oms2-error-log-green"
	ErrorAlias = "error-log"

	SystemIndex = "oms2-system-log-green"
	SystemAlias = "system-log"

	ErrorMessage  = "ошибка"
	SystemMessage = "системное сообщение"
)

type Elastic struct {
	cfg    config.Elastic
	client *elastic.Client
	health *elastic.ClusterHealthService
	zl     *zap.Logger
}

func NewElastic(cfg config.Elastic, zl *zap.Logger) *Elastic {
	return &Elastic{
		cfg: cfg,
		zl:  zl,
	}
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
		elastic.SetErrorLog(NewElasticErrorLogger(e.zl)),
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

	if e.cfg.SkipIndexCreation {
		return nil
	}

	err = e.CreateIndexesIfNotExists()
	if err != nil {
		return err
	}

	return nil
}

func (e *Elastic) Stop(_ context.Context) error {
	e.client.Stop()
	return nil
}

func (e *Elastic) IsReady(ctx context.Context) bool {
	resp, err := e.health.Do(ctx)
	if err != nil {
		e.zl.Error("cannot check elastic v7 health", zap.Error(err))
		return false
	}

	if resp.Status != elasticReadyHealthStatus {
		return false
	}

	return true
}

func (e *Elastic) CreateIndexesIfNotExists() (ok error) {

	ok = e.CreateIndexIfNotExists(ErrorIndex, ErrorAlias)
	if ok != nil {
		return ok
	}
	ok = e.CreateIndexIfNotExists(SystemIndex, SystemAlias)

	return ok
}

func (e *Elastic) CreateIndexIfNotExists(Index string, Alias string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	ok, err := e.client.IndexExists(Index).Do(ctx)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	mappingBody := getMappings()
	ctx, _ = context.WithTimeout(context.Background(), 10*time.Second)
	res, err := e.client.CreateIndex(Index).Body(mappingBody).Do(ctx)
	if err != nil {
		return err
	}
	if !res.Acknowledged {
		return errors.New("Index " + Index + " does not created")
	}
	err = e.AddAlias(Index, Alias)
	if err != nil {
		return err
	}
	return nil
}

func (e *Elastic) AddAlias(Index string, Alias string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	res, err := e.client.Alias().Add(Index, Alias).Do(ctx)
	if err != nil {
		return err
	}
	if !res.Acknowledged {
		return errors.New("Alias do not acknowledged")
	}
	return nil
}

func (e *Elastic) Create(ctx context.Context, message oms.LogMessage, id string, Index string) (string, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = opentracing.GlobalTracer().StartSpan("create", opentracing.ChildOf(span.Context()))
		defer span.Finish()
	}

	elasticCtx, _ := context.WithTimeout(ctx, 3*time.Second)
	idx := e.client.Index().Index(Index).BodyJson(message)
	if id != "" {
		idx.Id(id)
	}

	r, err := idx.Do(elasticCtx)
	if err != nil {
		return "", err
	}

	return r.Id, nil
}

func (e *Elastic) Get(ctx context.Context, id string, Alias string) (json.RawMessage, error) {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = opentracing.GlobalTracer().StartSpan("get", opentracing.ChildOf(span.Context()))
		defer span.Finish()
	}

	elasticCtx, _ := context.WithTimeout(ctx, 3*time.Second)
	result, err := e.client.Get().Index(Alias).Id(id).Do(elasticCtx)
	if err != nil {
		if elastic.IsNotFound(err) {
			return nil, err
		}
		return nil, err
	}

	return result.Source, nil
}

func (e *Elastic) Update(ctx context.Context, id string, schedule json.RawMessage, Index string) error {
	span := opentracing.SpanFromContext(ctx)
	if span != nil {
		span = opentracing.GlobalTracer().StartSpan("update", opentracing.ChildOf(span.Context()))
		defer span.Finish()
	}

	elasticCtx, _ := context.WithTimeout(ctx, 3*time.Second)
	if _, err := e.client.Index().Index(Index).Id(id).BodyJson(schedule).Do(elasticCtx); err != nil {
		return err
	}

	return nil
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

func getMappings() string {
	return `
{
    "settings": {
        "index": {
            "number_of_shards": "1",
            "number_of_replicas": "2"
        }
    },
    "mappings": {
        "properties": {
            "name": {
                "type": "keyword"
            },
			"node": {
                "type": "text"
            },
            "description": {
                "type": "text"
            },
			"kind": {
                "type": "text"
            },
			"timestamp": {
				"type": "text"
			}
        }
    }
}`
}
