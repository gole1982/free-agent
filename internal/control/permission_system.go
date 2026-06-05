package control

import (
	"fmt"
	"sync"
)

type Permission string

const (
	PermissionRead   Permission = "read"
	PermissionWrite  Permission = "write"
	PermissionExecute Permission = "execute"
	PermissionDelete Permission = "delete"
	PermissionAdmin  Permission = "admin"
)

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleUser       Role = "user"
	RoleGuest      Role = "guest"
	RoleDeveloper  Role = "developer"
)

type User struct {
	ID          string
	Name        string
	Role        Role
	Permissions []Permission
}

type PermissionSystem struct {
	roles       map[Role][]Permission
	users       map[string]*User
	permissions map[Permission]string
	mu          sync.RWMutex
}

func NewPermissionSystem() *PermissionSystem {
	return &PermissionSystem{
		roles: map[Role][]Permission{
			RoleAdmin: {
				PermissionRead,
				PermissionWrite,
				PermissionExecute,
				PermissionDelete,
				PermissionAdmin,
			},
			RoleDeveloper: {
				PermissionRead,
				PermissionWrite,
				PermissionExecute,
			},
			RoleUser: {
				PermissionRead,
				PermissionWrite,
			},
			RoleGuest: {
				PermissionRead,
			},
		},
		users: map[string]*User{},
		permissions: map[Permission]string{
			PermissionRead:   "Read access",
			PermissionWrite:  "Write access",
			PermissionExecute: "Execute commands",
			PermissionDelete: "Delete resources",
			PermissionAdmin:  "Admin access",
		},
	}
}

func (ps *PermissionSystem) AddUser(user *User) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.users[user.ID]; exists {
		return fmt.Errorf("user %s already exists", user.ID)
	}

	ps.users[user.ID] = user
	return nil
}

func (ps *PermissionSystem) GetUser(userID string) (*User, error) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	user, exists := ps.users[userID]
	if !exists {
		return nil, fmt.Errorf("user %s not found", userID)
	}

	return user, nil
}

func (ps *PermissionSystem) HasPermission(userID string, permission Permission) bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	user, exists := ps.users[userID]
	if !exists {
		return false
	}

	for _, p := range user.Permissions {
		if p == permission {
			return true
		}
	}

	rolePermissions, exists := ps.roles[user.Role]
	if !exists {
		return false
	}

	for _, p := range rolePermissions {
		if p == permission {
			return true
		}
	}

	return false
}

func (ps *PermissionSystem) CheckAccess(userID string, action string) (bool, string) {
	permissionMap := map[string]Permission{
		"read":    PermissionRead,
		"write":   PermissionWrite,
		"execute": PermissionExecute,
		"delete":  PermissionDelete,
		"admin":   PermissionAdmin,
	}

	permission, exists := permissionMap[action]
	if !exists {
		return false, fmt.Sprintf("Unknown action: %s", action)
	}

	if ps.HasPermission(userID, permission) {
		return true, fmt.Sprintf("Access granted for action: %s", action)
	}

	return false, fmt.Sprintf("Access denied for action: %s", action)
}

func (ps *PermissionSystem) ListRoles() []Role {
	roles := make([]Role, 0, len(ps.roles))
	for role := range ps.roles {
		roles = append(roles, role)
	}
	return roles
}

func (ps *PermissionSystem) GetRolePermissions(role Role) ([]Permission, error) {
	permissions, exists := ps.roles[role]
	if !exists {
		return nil, fmt.Errorf("role %s not found", role)
	}
	return permissions, nil
}
