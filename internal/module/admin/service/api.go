package service

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/repository"
)

type APIService struct {
	apiRepo *repository.ApiRepo
}

func NewAPIService() *APIService {
	return &APIService{apiRepo: repository.NewApiRepo()}
}

type CreateAPIRequest struct {
	Path        string `json:"path" binding:"required"`
	Method      string `json:"method" binding:"required"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

type UpdateAPIRequest struct {
	Path        string `json:"path"`
	Method      string `json:"method"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

func (s *APIService) Create(ctx context.Context, req *CreateAPIRequest) error {
	api := &model.SysAPI{
		Path:        req.Path,
		Method:      req.Method,
		Group:       req.Group,
		Description: req.Description,
	}
	return s.apiRepo.Create(ctx, api)
}

func (s *APIService) GetByID(ctx context.Context, id uint) (*model.SysAPI, error) {
	return s.apiRepo.GetByID(ctx, id)
}

func (s *APIService) Update(ctx context.Context, id uint, req *UpdateAPIRequest) error {
	api, err := s.apiRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "API 不存在")
	}
	if req.Path != "" {
		api.Path = req.Path
	}
	if req.Method != "" {
		api.Method = req.Method
	}
	if req.Group != "" {
		api.Group = req.Group
	}
	if req.Description != "" {
		api.Description = req.Description
	}
	return s.apiRepo.Update(ctx, api)
}

func (s *APIService) Delete(ctx context.Context, id uint) error {
	return s.apiRepo.Delete(ctx, id)
}

func (s *APIService) List(ctx context.Context, page, pageSize int) ([]model.SysAPI, int64, error) {
	return s.apiRepo.PageList(ctx, page, pageSize, &repository.QueryOptions{
		Order: []string{"`group` ASC", "id ASC"},
	})
}

func (s *APIService) ListAll(ctx context.Context) ([]model.SysAPI, error) {
	return s.apiRepo.ListAll(ctx)
}
