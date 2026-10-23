package service

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/kar1hsu/backplane/internal/model"
	"github.com/kar1hsu/backplane/internal/pkg/setting"
	"github.com/kar1hsu/backplane/internal/pkg/utils"
	"github.com/kar1hsu/backplane/internal/repository"
	"gorm.io/gorm"
)

// minPasswordFloor matches the binding tags below; security.password_min_length
// can only raise it.
const minPasswordFloor = 6

type UserService struct {
	userRepo *repository.UserRepo
	roleRepo *repository.RoleRepo
}

func NewUserService() *UserService {
	return &UserService{userRepo: repository.NewUserRepo(), roleRepo: repository.NewRoleRepo()}
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

func (s *UserService) Create(ctx context.Context, op Operator, req *CreateUserRequest) error {
	if err := checkPasswordLength(req.Password); err != nil {
		return err
	}
	roles, err := s.loadRoles(ctx, req.RoleIDs)
	if err != nil {
		return err
	}
	if hasSuperAdminRole(roles) && !op.IsSuperAdmin() {
		return forbidden("只有超级管理员可以分配超级管理员角色")
	}

	_, err = s.userRepo.GetByUsername(ctx, req.Username)
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
		if len(roles) > 0 {
			return s.userRepo.SetRoles(ctx, user.ID, uniqueIDs(req.RoleIDs))
		}
		return nil
	})
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*model.SysUser, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) Update(ctx context.Context, op Operator, id uint, req *UpdateUserRequest) error {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "用户不存在")
	}
	if hasSuperAdminRole(user.Roles) && !op.IsSuperAdmin() {
		return forbidden("只有超级管理员可以修改超级管理员账号")
	}

	// The edit form always sends status and role_ids, so compare with the
	// stored values: only real changes are guarded and revoke sessions.
	statusChanged := req.Status != nil && *req.Status != user.Status
	rolesChanged := false
	if req.RoleIDs != nil {
		newRoles, err := s.loadRoles(ctx, req.RoleIDs)
		if err != nil {
			return err
		}
		currentIDs := make([]uint, 0, len(user.Roles))
		for _, r := range user.Roles {
			currentIDs = append(currentIDs, r.ID)
		}
		rolesChanged = !sameIDs(currentIDs, req.RoleIDs)
		if rolesChanged && hasSuperAdminRole(newRoles) && !op.IsSuperAdmin() {
			return forbidden("只有超级管理员可以分配超级管理员角色")
		}
	}
	if id == op.UserID && (statusChanged || rolesChanged) {
		return forbidden("不能修改自己的状态或角色")
	}
	if req.Password != "" {
		if err := checkPasswordLength(req.Password); err != nil {
			return err
		}
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
	if statusChanged {
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
	if req.Password != "" || statusChanged || rolesChanged {
		user.TokenVersion++
	}

	return repository.Transaction(ctx, func(ctx context.Context) error {
		if err := s.userRepo.Update(ctx, user); err != nil {
			return err
		}
		if rolesChanged {
			return s.userRepo.SetRoles(ctx, id, uniqueIDs(req.RoleIDs))
		}
		return nil
	})
}

func (s *UserService) Delete(ctx context.Context, id, currentUserID uint) error {
	if id == currentUserID {
		return forbidden("不能删除当前登录用户")
	}
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return notFoundOr(err, "用户不存在")
	}
	if hasSuperAdminRole(user.Roles) {
		return forbidden("不能删除超级管理员")
	}
	return s.userRepo.Delete(ctx, id)
}

// loadRoles resolves role IDs and rejects unknown ones, which GORM's
// Association.Replace would otherwise insert as blank roles.
func (s *UserService) loadRoles(ctx context.Context, ids []uint) ([]model.SysRole, error) {
	ids = uniqueIDs(ids)
	roles, err := s.roleRepo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(roles) != len(ids) {
		return nil, errors.New("角色不存在")
	}
	return roles, nil
}

func checkPasswordLength(password string) error {
	minLen := setting.GetInt(setting.PasswordMinLengthKey)
	if minLen < minPasswordFloor {
		minLen = minPasswordFloor
	}
	if utf8.RuneCountInString(password) < minLen {
		return fmt.Errorf("密码长度不能少于 %d 位", minLen)
	}
	return nil
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
