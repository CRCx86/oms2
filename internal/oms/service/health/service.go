package health

import "oms2/internal/oms"

type Service struct {
	cfg *oms.Config
}

func NewService(cfg *oms.Config) *Service {
	return &Service{cfg: cfg}
}

func (s *Service) Health() map[string]string {
	return map[string]string{"status": "ok"}
}
