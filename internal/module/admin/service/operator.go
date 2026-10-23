package service

import (
	"slices"

	"github.com/kar1hsu/backplane/internal/model"
)

// Operator is the authenticated admin performing a request. Route-level RBAC
// only decides whether an endpoint may be called at all; services use the
// operator for rules RBAC cannot express, such as who may grant the
// super-admin role or whether someone is editing their own permissions.
type Operator struct {
	UserID    uint
	RoleCodes []string
}

func (o Operator) IsSuperAdmin() bool {
	return slices.Contains(o.RoleCodes, model.SuperAdminRoleCode)
}

func (o Operator) holdsRole(code string) bool {
	return slices.Contains(o.RoleCodes, code)
}

func requireSuperAdmin(op Operator, msg string) error {
	if op.IsSuperAdmin() {
		return nil
	}
	return forbidden(msg)
}

func hasSuperAdminRole(roles []model.SysRole) bool {
	return slices.ContainsFunc(roles, func(r model.SysRole) bool {
		return r.Code == model.SuperAdminRoleCode
	})
}

// sameIDs reports whether two ID lists hold the same set, ignoring order and
// duplicates.
func sameIDs(a, b []uint) bool {
	set := make(map[uint]struct{}, len(a))
	for _, id := range a {
		set[id] = struct{}{}
	}
	seen := make(map[uint]struct{}, len(b))
	for _, id := range b {
		if _, ok := set[id]; !ok {
			return false
		}
		seen[id] = struct{}{}
	}
	return len(seen) == len(set)
}

// uniqueIDs drops duplicates while keeping the first-seen order.
func uniqueIDs(ids []uint) []uint {
	seen := make(map[uint]struct{}, len(ids))
	out := make([]uint, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
