package service

import (
	"context"
	"errors"

	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"github.com/kar1hsu/backplane/internal/repository"
	"gorm.io/gorm"
)

type UserService struct {
	userRepo *repository.UserRepo
}

func NewUserService() *UserService {
	return &UserService{userRepo: repository.NewUserRepo()}
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6,max=72"`
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Status   *int8  `json:"status"`
	RoleIDs  []uint `json:"role_ids"`
}

type UpdateUserRequest struct {
	Nickname string `json:"nickname"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Avatar   string `json:"avatar"`
	Status   *int8  `json:"status"`
	Password string `json:"password" binding:"omitempty,min=6,max=72"`
	RoleIDs  []uint `json:"role_ids"`
}

func (s *UserService) Create(ctx context.Context, req *CreateUserRequest) error {
	_, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil {
		return errors.New("用户名已存在")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err // 真实 DB 错误，不当作"可创建"
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		return errors.New("密码加密失败")
	}

	status := int8(1)
	if req.Status != nil {
		status = *req.Status
	}

	user := &model.SysUser{
		Username: req.Username,
		Password: hashed,
		Nickname: req.Nickname,
		Email:    req.Email,
		Phone:    req.Phone,
		Status:   status,
	}

	// 创建用户 + 分配角色 在同一事务内（中途失败整体回滚）
	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Create(ctx, user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errors.New("用户名已存在") // 唯一索引兜底（堵 TOCTOU 竞态）
			}
			return err
		}
		if len(req.RoleIDs) > 0 {
			return s.userRepo.SetRoles(ctx, user.ID, req.RoleIDs)
		}
		return nil
	})
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*model.SysUser, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, id uint, req *UpdateUserRequest) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "用户不存在")
	}

	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Status != nil {
		user.Status = *req.Status
	}
	if req.Password != "" {
		hashed, err := utils.HashPassword(req.Password)
		if err != nil {
			return errors.New("密码加密失败")
		}
		user.Password = hashed
	}
	// 改密 / 改状态 / 改角色都使该用户已签发的 token 失效（会话撤销）
	if req.Password != "" || req.Status != nil || req.RoleIDs != nil {
		user.TokenVersion++
	}

	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}
		if req.RoleIDs != nil {
			return s.userRepo.SetRoles(ctx, id, req.RoleIDs)
		}
		return nil
	})
}

func (s *UserService) Delete(ctx context.Context, id, currentUserID uint) error {
	if id == currentUserID {
		return errors.New("不能删除当前登录用户")
	}
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "用户不存在")
	}
	for _, r := range user.Roles {
		if r.Code == model.SuperAdminRoleCode {
			return errors.New("不能删除超级管理员")
		}
	}
	return s.userRepo.Delete(ctx, id)
}

func (s *UserService) List(ctx context.Context, page, pageSize int) ([]model.SysUser, int64, error) {
	return s.userRepo.PageList(ctx, page, pageSize, &repository.QueryOptions{
		Order:    []string{"id DESC"},
		Preloads: []string{"Roles"},
	})
}

func (s *UserService) GetProfile(ctx context.Context, id uint) (*model.SysUser, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, notFoundOr(err, "用户不存在")
	}
	return user, nil
}
