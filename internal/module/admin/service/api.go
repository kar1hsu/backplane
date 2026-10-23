package service

import (
	"context"
	"strings"

	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/repository"
)

type APIService struct {
	apiRepo *repository.ApiRepo
}

func NewAPIService() *APIService {
	return &APIService{apiRepo: repository.NewApiRepo()}
}

// The RBAC matcher compares methods exactly (r.act == p.act), so methods are
// stored upper-case and limited to what the router serves.
type CreateAPIRequest struct {
	Path        string `json:"path" binding:"required,startswith=/"`
	Method      string `json:"method" binding:"required,oneof=GET POST PUT DELETE PATCH get post put delete patch"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

type UpdateAPIRequest struct {
	Path        string `json:"path" binding:"omitempty,startswith=/"`
	Method      string `json:"method" binding:"omitempty,oneof=GET POST PUT DELETE PATCH get post put delete patch"`
	Group       string `json:"group"`
	Description string `json:"description"`
}

// API records are the path/method pairs that menus grant, so editing them
// changes what roles may call: super-admin only.
const apiWriteForbiddenMsg = "只有超级管理员可以修改 API 定义"

func (s *APIService) Create(ctx context.Context, op Operator, req *CreateAPIRequest) error {
	if err := requireSuperAdmin(op, apiWriteForbiddenMsg); err != nil {
		return err
	}
	api := &model.SysAPI{
		Path:        req.Path,
		Method:      strings.ToUpper(req.Method),
		Group:       req.Group,
		Description: req.Description,
	}
	return s.apiRepo.Create(ctx, api)
}

func (s *APIService) GetByID(ctx context.Context, id uint) (*model.SysAPI, error) {
	return s.apiRepo.GetByID(ctx, id)
}

func (s *APIService) Update(ctx context.Context, op Operator, id uint, req *UpdateAPIRequest) error {
	if err := requireSuperAdmin(op, apiWriteForbiddenMsg); err != nil {
		return err
	}
	api, err := s.apiRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "API 不存在")
	}
	oldPath, oldMethod := api.Path, api.Method
	if req.Path != "" {
		api.Path = req.Path
	}
	if req.Method != "" {
		api.Method = strings.ToUpper(req.Method)
	}
	if req.Group != "" {
		api.Group = req.Group
	}
	if req.Description != "" {
		api.Description = req.Description
	}
	if err := s.apiRepo.Update(ctx, api); err != nil {
		return err
	}
	if api.Path == oldPath && api.Method == oldMethod {
		return nil
	}
	// Roles were granted the old path/method; re-grant what their menus bind.
	return s.rebuildPoliciesForAPI(ctx, id)
}

// rebuildPoliciesForAPI re-syncs every role whose menus bind the API.
func (s *APIService) rebuildPoliciesForAPI(ctx context.Context, apiID uint) error {
	menuIDs, err := repository.NewMenuRepo().IDsByAPI(ctx, apiID)
	if err != nil {
		return err
	}
	return rebuildPoliciesForMenus(ctx, menuIDs)
}

func (s *APIService) Delete(ctx context.Context, op Operator, id uint) error {
	if err := requireSuperAdmin(op, apiWriteForbiddenMsg); err != nil {
		return err
	}
	// Collect the affected menus before the delete unbinds them.
	menuIDs, err := repository.NewMenuRepo().IDsByAPI(ctx, id)
	if err != nil {
		return err
	}
	if err := s.apiRepo.Delete(ctx, id); err != nil {
		return err
	}
	return rebuildPoliciesForMenus(ctx, menuIDs)
}

func (s *APIService) List(ctx context.Context, page, pageSize int) ([]model.SysAPI, int64, error) {
	return s.apiRepo.PageList(ctx, page, pageSize, &repository.QueryOptions{
		OrderColumns: repository.GroupOrder,
	})
}

func (s *APIService) ListAll(ctx context.Context) ([]model.SysAPI, error) {
	return s.apiRepo.ListAll(ctx)
}
