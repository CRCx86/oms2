package log

import (
	"context"
	"go.uber.org/zap"
	"oms2/internal/pkg/util"

	"oms2/internal/oms"
	v7 "oms2/internal/pkg/storage/elastic/v7"
)

type Service struct {
	zl      *zap.Logger
	cfg     *oms.Config
	storage *v7.Elastic
}

func NewService(cfg *oms.Config, storage *v7.Elastic, zl *zap.Logger) *Service {
	return &Service{
		zl:      zl,
		cfg:     cfg,
		storage: storage,
	}
}

func (s *Service) LogMessage(
	c context.Context,
	messageType string,
	message string,
	indexElastic string,
	data map[string]interface{}) {

	go func() {
		_, err := s.storage.Create(c,
			util.MessageToExternalLog(
				data,
				messageType,
				message),
			"",
			indexElastic)
		s.zl.Sugar().Info(message)
		if err != nil {
			s.zl.Sugar().Info(err)
		}
	}()

}
