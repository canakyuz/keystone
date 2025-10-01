package user

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)

	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "tenant-001", user.TenantID)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.Equal(t, UserStatusPending, user.Status)
	assert.False(t, user.EmailVerified)
	assert.NotEmpty(t, user.Password)
	assert.NotEqual(t, "SecurePass123!", user.Password, "Password should be hashed")
	assert.False(t, user.CreatedAt.IsZero())
	assert.False(t, user.UpdatedAt.IsZero())
}

func TestHashPassword(t *testing.T) {
	password := "SecurePass123!"

	hash, err := HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
	assert.True(t, len(hash) > 0)
}

func TestUser_CheckPassword(t *testing.T) {
	password := "SecurePass123!"
	user, err := New("tenant-001", "test@example.com", password, "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "correct password",
			password: "SecurePass123!",
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "WrongPassword",
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := user.CheckPassword(tt.password)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestUser_UpdatePassword(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "OldPass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)
	oldHash := user.Password

	err = user.UpdatePassword("NewSecurePass123!")
	require.NoError(t, err)
	assert.NotEqual(t, oldHash, user.Password)
	assert.True(t, user.CheckPassword("NewSecurePass123!"))
}

func TestUser_Suspend(t *testing.T) {
	user := &User{
		ID:     "user-001",
		Status: UserStatusActive,
	}

	err := user.Suspend("Policy violation")
	require.NoError(t, err)
	assert.Equal(t, UserStatusSuspended, user.Status)
}

func TestUser_Activate(t *testing.T) {
	user := &User{
		ID:     "user-001",
		Status: UserStatusSuspended,
	}

	err := user.Activate()
	require.NoError(t, err)
	assert.Equal(t, UserStatusActive, user.Status)
}

func TestUser_VerifyEmail(t *testing.T) {
	user := &User{
		ID:            "user-001",
		EmailVerified: false,
	}

	err := user.VerifyEmail()
	require.NoError(t, err)
	assert.True(t, user.EmailVerified)
	assert.NotNil(t, user.EmailVerifiedAt)
	assert.WithinDuration(t, time.Now(), *user.EmailVerifiedAt, time.Second)
}

func TestUser_UpdateLastLogin(t *testing.T) {
	user := &User{
		ID:          "user-001",
		LastLoginAt: nil,
	}

	user.UpdateLastLogin()

	assert.NotNil(t, user.LastLoginAt)
	assert.WithinDuration(t, time.Now(), *user.LastLoginAt, time.Second)
}

func TestUserRole_GetPermissions(t *testing.T) {
	tests := []struct {
		name             string
		role             UserRole
		shouldHaveRead   bool
		shouldHaveWrite  bool
		shouldHaveManage bool
	}{
		{
			name:             "owner has all permissions",
			role:             RoleOwner,
			shouldHaveRead:   true,
			shouldHaveWrite:  true,
			shouldHaveManage: true,
		},
		{
			name:             "admin has management permissions",
			role:             RoleAdmin,
			shouldHaveRead:   true,
			shouldHaveWrite:  true,
			shouldHaveManage: true,
		},
		{
			name:             "editor has edit permissions",
			role:             RoleEditor,
			shouldHaveRead:   true,
			shouldHaveWrite:  true,
			shouldHaveManage: false,
		},
		{
			name:             "viewer has read-only permissions",
			role:             RoleViewer,
			shouldHaveRead:   true,
			shouldHaveWrite:  false,
			shouldHaveManage: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := tt.role.GetPermissions()

			assert.NotEmpty(t, perms)

			if tt.shouldHaveRead {
				assert.Contains(t, perms, "content:read")
			}

			if tt.shouldHaveWrite {
				assert.Contains(t, perms, "content:write")
			}

			if tt.shouldHaveManage {
				assert.Contains(t, perms, "users:manage")
			} else {
				assert.NotContains(t, perms, "users:manage")
			}
		})
	}
}

func TestUserStatus_Validation(t *testing.T) {
	validStatuses := []UserStatus{UserStatusActive, UserStatusSuspended, UserStatusPending}

	for _, status := range validStatuses {
		assert.NotEmpty(t, status, "Status should not be empty")
	}
}

func TestUserRole_Validation(t *testing.T) {
	validRoles := []UserRole{RoleOwner, RoleAdmin, RoleEditor, RoleViewer}

	for _, role := range validRoles {
		assert.NotEmpty(t, role, "Role should not be empty")
		perms := role.GetPermissions()
		assert.NotEmpty(t, perms, "Permissions should not be empty")
	}
}
