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

// 🎓 ADDITIONAL COVERAGE TESTS: Missing method coverage

func TestUser_IsPending(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	// New users are pending by default
	assert.True(t, user.IsPending())

	// Activate user
	user.Activate()
	assert.False(t, user.IsPending())
}

func TestUser_HasRole(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	assert.True(t, user.HasRole(RoleAdmin))
	assert.False(t, user.HasRole(RoleOwner))
	assert.False(t, user.HasRole(RoleEditor))
}

func TestUser_IsOwner(t *testing.T) {
	owner, err := New("tenant-001", "owner@example.com", "SecurePass123!", "Owner", "User", RoleOwner)
	require.NoError(t, err)
	admin, err := New("tenant-001", "admin@example.com", "SecurePass123!", "Admin", "User", RoleAdmin)
	require.NoError(t, err)

	assert.True(t, owner.IsOwner())
	assert.False(t, admin.IsOwner())
}

func TestUser_IsAdmin(t *testing.T) {
	admin, err := New("tenant-001", "admin@example.com", "SecurePass123!", "Admin", "User", RoleAdmin)
	require.NoError(t, err)
	editor, err := New("tenant-001", "editor@example.com", "SecurePass123!", "Editor", "User", RoleEditor)
	require.NoError(t, err)

	assert.True(t, admin.IsAdmin())
	assert.False(t, editor.IsAdmin())
}

func TestUser_CanEdit(t *testing.T) {
	owner, err := New("tenant-001", "owner@example.com", "SecurePass123!", "Owner", "User", RoleOwner)
	require.NoError(t, err)
	admin, err := New("tenant-001", "admin@example.com", "SecurePass123!", "Admin", "User", RoleAdmin)
	require.NoError(t, err)
	editor, err := New("tenant-001", "editor@example.com", "SecurePass123!", "Editor", "User", RoleEditor)
	require.NoError(t, err)
	viewer, err := New("tenant-001", "viewer@example.com", "SecurePass123!", "Viewer", "User", RoleViewer)
	require.NoError(t, err)

	assert.True(t, owner.CanEdit())
	assert.True(t, admin.CanEdit())
	assert.True(t, editor.CanEdit())
	assert.False(t, viewer.CanEdit())
}

func TestUser_CanView(t *testing.T) {
	owner, err := New("tenant-001", "owner@example.com", "SecurePass123!", "Owner", "User", RoleOwner)
	require.NoError(t, err)
	viewer, err := New("tenant-001", "viewer@example.com", "SecurePass123!", "Viewer", "User", RoleViewer)
	require.NoError(t, err)

	// All roles can view
	assert.True(t, owner.CanView())
	assert.True(t, viewer.CanView())
}

func TestUser_UpdateProfile(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	oldUpdatedAt := user.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	user.UpdateProfile("Jane", "Smith", "+1234567890", "Europe/Istanbul", "tr")

	assert.Equal(t, "Jane", user.FirstName)
	assert.Equal(t, "Smith", user.LastName)
	assert.Equal(t, "+1234567890", user.Phone)
	assert.Equal(t, "Europe/Istanbul", user.Timezone)
	assert.Equal(t, "tr", user.Locale)
	assert.True(t, user.UpdatedAt.After(oldUpdatedAt))
}

func TestUser_UpdateAvatar(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	oldUpdatedAt := user.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	avatarURL := "https://example.com/avatar.jpg"
	user.UpdateAvatar(avatarURL)

	assert.Equal(t, avatarURL, user.Avatar)
	assert.True(t, user.UpdatedAt.After(oldUpdatedAt))
}

func TestUser_Activate_ErrorCases(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	// First activation should succeed
	err = user.Activate()
	assert.NoError(t, err)
	assert.Equal(t, UserStatusActive, user.Status)

	// Second activation should fail (already active)
	err = user.Activate()
	assert.Error(t, err)
	assert.Equal(t, ErrUserAlreadyActive, err)
}

func TestUser_Suspend_ErrorCases(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)
	user.Activate()

	// First suspension should succeed
	err = user.Suspend("Policy violation")
	assert.NoError(t, err)
	assert.Equal(t, UserStatusSuspended, user.Status)

	// Second suspension should fail (already suspended)
	err = user.Suspend("Another reason")
	assert.Error(t, err)
	assert.Equal(t, ErrUserAlreadySuspended, err)
}

func TestUser_VerifyEmail_ErrorCases(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	// First verification should succeed
	err = user.VerifyEmail()
	assert.NoError(t, err)
	assert.True(t, user.EmailVerified)
	assert.NotNil(t, user.EmailVerifiedAt)

	// Second verification should fail (already verified)
	err = user.VerifyEmail()
	assert.Error(t, err)
	assert.Equal(t, ErrEmailAlreadyVerified, err)
}

func TestUser_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() *User
		expectError bool
		errorType   error
	}{
		{
			name: "missing ID",
			setup: func() *User {
				u, _ := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
				u.ID = ""
				return u
			},
			expectError: true,
			errorType:   ErrInvalidUserID,
		},
		{
			name: "missing tenant ID",
			setup: func() *User {
				u, _ := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
				u.TenantID = ""
				return u
			},
			expectError: true,
			errorType:   ErrTenantIDRequired,
		},
		{
			name: "missing email",
			setup: func() *User {
				u, _ := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
				u.Email = ""
				return u
			},
			expectError: true,
			errorType:   ErrEmailRequired,
		},
		{
			name: "email too long",
			setup: func() *User {
				u, _ := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
				u.Email = string(make([]byte, 256)) + "@example.com"
				return u
			},
			expectError: true,
			errorType:   ErrInvalidEmail,
		},
		{
			name: "missing first name",
			setup: func() *User {
				u, _ := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
				u.FirstName = ""
				return u
			},
			expectError: true,
			errorType:   ErrFirstNameRequired,
		},
		{
			name: "missing last name",
			setup: func() *User {
				u, _ := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
				u.LastName = ""
				return u
			},
			expectError: true,
			errorType:   ErrLastNameRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := tt.setup()
			err := user.Validate()

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorType, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUser_UpdateRole(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleViewer)
	require.NoError(t, err)

	assert.Equal(t, RoleViewer, user.Role)

	user.UpdateRole(RoleAdmin)
	assert.Equal(t, RoleAdmin, user.Role)
}

func TestUser_RecordLogin(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	assert.Nil(t, user.LastLoginAt)

	user.RecordLogin()

	assert.NotNil(t, user.LastLoginAt)
	assert.WithinDuration(t, time.Now(), *user.LastLoginAt, time.Second)
}

func TestUser_TwoFactor(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	assert.False(t, user.TwoFactorEnabled)

	// Enable 2FA
	user.EnableTwoFactor()
	assert.True(t, user.TwoFactorEnabled)

	// Disable 2FA
	user.DisableTwoFactor()
	assert.False(t, user.TwoFactorEnabled)
}

func TestUser_Preferences(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	// Get non-existent preference
	value, exists := user.GetPreference("theme")
	assert.Nil(t, value)
	assert.False(t, exists)

	// Set preference
	user.UpdatePreference("theme", "dark")
	value, exists = user.GetPreference("theme")
	assert.Equal(t, "dark", value)
	assert.True(t, exists)

	// Update preference
	user.UpdatePreference("theme", "light")
	value, exists = user.GetPreference("theme")
	assert.Equal(t, "light", value)
	assert.True(t, exists)
}

func TestUser_FullName(t *testing.T) {
	user, err := New("tenant-001", "test@example.com", "SecurePass123!", "John", "Doe", RoleAdmin)
	require.NoError(t, err)

	assert.Equal(t, "John Doe", user.FullName())
}

func TestUserRole_HasPermission(t *testing.T) {
	// Admin has write permission
	assert.True(t, RoleAdmin.HasPermission("content:write"))

	// Viewer doesn't have write permission
	assert.False(t, RoleViewer.HasPermission("content:write"))

	// Viewer has read permission
	assert.True(t, RoleViewer.HasPermission("content:read"))

	// Owner has all permissions
	assert.True(t, RoleOwner.HasPermission("users:manage"))
	assert.True(t, RoleOwner.HasPermission("content:write"))
}

func TestUserStatus_IsValid(t *testing.T) {
	assert.True(t, UserStatusActive.IsValid())
	assert.True(t, UserStatusSuspended.IsValid())
	assert.True(t, UserStatusPending.IsValid())
	assert.False(t, UserStatus("invalid").IsValid())
}

func TestUserRole_IsValid(t *testing.T) {
	assert.True(t, RoleOwner.IsValid())
	assert.True(t, RoleAdmin.IsValid())
	assert.True(t, RoleEditor.IsValid())
	assert.True(t, RoleViewer.IsValid())
	assert.False(t, UserRole("invalid").IsValid())
}
