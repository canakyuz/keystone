package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTenant(t *testing.T) {
	tests := []struct {
		name    string
		tnName  string
		slug    string
		email   string
		plan    SubscriptionPlan
		wantErr bool
	}{
		{
			name:    "valid tenant creation",
			tnName:  "Test Company",
			slug:    "test-company",
			email:   "test@example.com",
			plan:    PlanPro,
			wantErr: false,
		},
		{
			name:    "empty name should fail",
			tnName:  "",
			slug:    "test",
			email:   "test@example.com",
			plan:    PlanFree,
			wantErr: true,
		},
		{
			name:    "empty slug should fail",
			tnName:  "Test",
			slug:    "",
			email:   "test@example.com",
			plan:    PlanFree,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant, err := New(tt.tnName, tt.slug, tt.email, tt.plan)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, tenant)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, tenant)
				assert.Equal(t, tt.tnName, tenant.Name)
				assert.Equal(t, tt.slug, tenant.Slug)
				assert.Equal(t, tt.email, tenant.Email)
				assert.Equal(t, tt.plan, tenant.Plan)
				assert.False(t, tenant.CreatedAt.IsZero())
				assert.False(t, tenant.UpdatedAt.IsZero())
			}
		})
	}
}

func TestTenant_Activate(t *testing.T) {
	tenant := &Tenant{
		ID:     "test-id",
		Status: TenantStatusSuspended,
	}

	err := tenant.Activate()
	require.NoError(t, err)
	assert.Equal(t, TenantStatusActive, tenant.Status)
}

func TestTenant_Suspend(t *testing.T) {
	tests := []struct {
		name   string
		status TenantStatus
		reason string
	}{
		{
			name:   "suspend active tenant",
			status: TenantStatusActive,
			reason: "Payment failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{
				ID:     "test-id",
				Status: tt.status,
			}

			err := tenant.Suspend(tt.reason)
			require.NoError(t, err)
			assert.Equal(t, TenantStatusSuspended, tenant.Status)
		})
	}
}

func TestTenant_UpgradePlan(t *testing.T) {
	tests := []struct {
		name        string
		currentPlan SubscriptionPlan
		newPlan     SubscriptionPlan
	}{
		{
			name:        "upgrade from free to starter",
			currentPlan: PlanFree,
			newPlan:     PlanStarter,
		},
		{
			name:        "upgrade from starter to pro",
			currentPlan: PlanStarter,
			newPlan:     PlanPro,
		},
		{
			name:        "upgrade from pro to enterprise",
			currentPlan: PlanPro,
			newPlan:     PlanEnterprise,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{
				ID:   "test-id",
				Plan: tt.currentPlan,
			}

			err := tenant.UpgradePlan(tt.newPlan)
			require.NoError(t, err)
			assert.Equal(t, tt.newPlan, tenant.Plan)
		})
	}
}

func TestTenant_SetCustomDomain(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{
			name:   "valid domain",
			domain: "example.com",
		},
		{
			name:   "valid subdomain",
			domain: "app.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tenant := &Tenant{
				ID: "test-id",
			}

			err := tenant.SetCustomDomain(tt.domain)
			require.NoError(t, err)
			assert.Equal(t, tt.domain, tenant.CustomDomain)
			assert.False(t, tenant.CustomDomainVerified)
		})
	}
}

func TestTenant_VerifyCustomDomain(t *testing.T) {
	tenant := &Tenant{
		ID:           "test-id",
		CustomDomain: "example.com",
	}

	err := tenant.VerifyCustomDomain()
	require.NoError(t, err)
	assert.True(t, tenant.CustomDomainVerified)
	assert.NotNil(t, tenant.CustomDomainVerifiedAt)
}

func TestSubscriptionPlan_GetFeatureLimits(t *testing.T) {
	tests := []struct {
		name          string
		plan          SubscriptionPlan
		expectedUsers int
		expectedSites int
		customDomain  bool
		apiAccess     bool
	}{
		{
			name:          "free plan limits",
			plan:          PlanFree,
			expectedUsers: 1,
			expectedSites: 1,
			customDomain:  false,
			apiAccess:     false,
		},
		{
			name:          "starter plan limits",
			plan:          PlanStarter,
			expectedUsers: 5,
			expectedSites: 3,
			customDomain:  false,
			apiAccess:     true,
		},
		{
			name:          "pro plan limits",
			plan:          PlanPro,
			expectedUsers: 20,
			expectedSites: 10,
			customDomain:  true,
			apiAccess:     true,
		},
		{
			name:          "enterprise plan limits",
			plan:          PlanEnterprise,
			expectedUsers: -1, // unlimited
			expectedSites: -1, // unlimited
			customDomain:  true,
			apiAccess:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := tt.plan.GetFeatureLimits()

			assert.Equal(t, tt.expectedUsers, limits.MaxUsers)
			assert.Equal(t, tt.expectedSites, limits.MaxWebsites)
			assert.Equal(t, tt.customDomain, limits.CustomDomain)
			assert.Equal(t, tt.apiAccess, limits.APIAccess)
		})
	}
}

func TestTenantStatus_Validation(t *testing.T) {
	validStatuses := []TenantStatus{TenantStatusActive, TenantStatusSuspended, TenantStatusTrial}

	for _, status := range validStatuses {
		assert.NotEmpty(t, status, "Status should not be empty")
		assert.True(t, status.IsValid())
	}
}

func TestSubscriptionPlan_Validation(t *testing.T) {
	validPlans := []SubscriptionPlan{PlanFree, PlanStarter, PlanPro, PlanEnterprise}

	for _, plan := range validPlans {
		assert.NotEmpty(t, plan, "Plan should not be empty")
		assert.True(t, plan.IsValid())
		limits := plan.GetFeatureLimits()
		assert.NotEqual(t, FeatureLimits{}, limits, "Feature limits should not be empty")
	}
}
