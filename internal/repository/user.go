package repository

import (
	"context"

	"github.com/kar1hsu/backplane/internal/model"
	"gorm.io/gorm"
)

type UserRepo struct {
	BaseRepo[model.SysUser]
}

func NewUserRepo() *UserRepo {
	return &UserRepo{}
}

// GetByID overrides the generic version to preload Roles.
func (d *UserRepo) GetByID(ctx context.Context, id uint) (*model.SysUser, error) {
	return d.BaseRepo.GetByID(ctx, id, "Roles")
}

func (d *UserRepo) GetByUsername(ctx context.Context, username string) (*model.SysUser, error) {
	var user model.SysUser
	if err := dbFrom(ctx).Preload("Roles").Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// AuthState is what AdminAuth needs to authorize a request: the live session
// version, the account status and the codes of the user's enabled roles.
type AuthState struct {
	TokenVersion int
	Status       int8
	RoleCodes    []string
}

// GetAuthState loads the user's current auth state from the DB, so revoked
// sessions, disabled accounts and disabled or deleted roles take effect at once
// instead of trusting whatever was true when the token was issued.
func (d *UserRepo) GetAuthState(ctx context.Context, id uint) (*AuthState, error) {
	var user model.SysUser
	if err := dbFrom(ctx).Select("id", "status", "token_version").First(&user, id).Error; err != nil {
		return nil, err
	}
	var codes []string
	if err := dbFrom(ctx).Model(&model.SysRole{}).
		Joins("JOIN sys_user_role ON sys_user_role.sys_role_id = sys_role.id").
		Where("sys_user_role.sys_user_id = ? AND sys_role.status = ?", id, 1).
		Pluck("sys_role.code", &codes).Error; err != nil {
		return nil, err
	}
	return &AuthState{TokenVersion: user.TokenVersion, Status: user.Status, RoleCodes: codes}, nil
}

// Update writes only base columns; Roles is managed by SetRoles.
func (d *UserRepo) Update(ctx context.Context, user *model.SysUser) error {
	return d.BaseRepo.Update(ctx, user, "Nickname", "Email", "Phone", "Avatar", "Status", "Password", "TokenVersion")
}

// Delete soft-deletes the user, clears its role associations (sys_user_role),
// and mangles the username so the unique index is freed — letting the same
// username be reused later. The soft-deleted row is kept for audit.
func (d *UserRepo) Delete(ctx context.Context, id uint) error {
	return Transaction(ctx, func(ctx context.Context) error {
		if err := dbFrom(ctx).Model(&model.SysUser{}).Where("id = ?", id).
			Update("username", gorm.Expr("CONCAT('del#', id, '#', LEFT(username, 40))")).Error; err != nil {
			return err
		}
		return dbFrom(ctx).Select("Roles").Delete(&model.SysUser{ID: id}).Error
	})
}

func (d *UserRepo) SetRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	user := &model.SysUser{ID: userID}
	var roles []model.SysRole
	for _, id := range roleIDs {
		roles = append(roles, model.SysRole{ID: id})
	}
	return dbFrom(ctx).Model(user).Association("Roles").Replace(roles)
}
