package service

import (
	"errors"
	"strings"

	"skoll2/backend/internal/store"
)

type SampleHelloService struct {
	pluginSvc *PluginService
	store     *store.PluginStore
}

type CreateSampleHelloRecordRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type UpdateSampleHelloRecordRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

func NewSampleHelloService(pluginSvc *PluginService, pluginStore *store.PluginStore) *SampleHelloService {
	return &SampleHelloService{
		pluginSvc: pluginSvc,
		store:     pluginStore,
	}
}

func (s *SampleHelloService) ListRecords() ([]store.SampleHelloRecord, error) {
	if !s.pluginSvc.IsEnabled("sample-hello") {
		return nil, errors.New("sample-hello plugin is not enabled")
	}
	return s.store.ListSampleHelloRecords()
}

func (s *SampleHelloService) CreateRecord(req CreateSampleHelloRecordRequest) (store.SampleHelloRecord, error) {
	if !s.pluginSvc.IsEnabled("sample-hello") {
		return store.SampleHelloRecord{}, errors.New("sample-hello plugin is not enabled")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return store.SampleHelloRecord{}, errors.New("title is required")
	}

	return s.store.CreateSampleHelloRecord(title, req.Content)
}

func (s *SampleHelloService) UpdateRecord(id uint, req UpdateSampleHelloRecordRequest) (store.SampleHelloRecord, error) {
	if !s.pluginSvc.IsEnabled("sample-hello") {
		return store.SampleHelloRecord{}, errors.New("sample-hello plugin is not enabled")
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return store.SampleHelloRecord{}, errors.New("title is required")
	}

	return s.store.UpdateSampleHelloRecord(id, title, req.Content)
}

func (s *SampleHelloService) DeleteRecord(id uint) error {
	if !s.pluginSvc.IsEnabled("sample-hello") {
		return errors.New("sample-hello plugin is not enabled")
	}
	return s.store.DeleteSampleHelloRecord(id)
}
