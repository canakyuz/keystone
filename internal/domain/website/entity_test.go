package website

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 🎓 EDUCATIONAL NOTE: Go Testing Best Practices
//
// Bu test dosyası aşağıdaki Go konseptlerini kullanır:
//
// 1. Table-Driven Tests (test senaryolarını tablo olarak organize etme)
//    - Birden fazla test case'ini tek bir test fonksiyonunda çalıştırma
//    - Her case için: name (test adı), input (girdi), expected (beklenen)
//
// 2. Test Helpers (yardımcı fonksiyonlar)
//    - createTestWebsite(): Hızlı test website oluşturma
//    - Test kodunu DRY yapma (Don't Repeat Yourself)
//
// 3. Assertions vs Requirements
//    - assert: Test devam eder (failed olur ama çalışır)
//    - require: Test durur (critical error)
//
// 4. Test Isolation
//    - Her test bağımsızdır, başkaları etkilemez
//    - Test sırası önemli değildir

// ═════════════════════════════════════════════════════════════
// TEST HELPERS (Yardımcı Fonksiyonlar)
// ═════════════════════════════════════════════════════════════

// createTestWebsite creates a valid website for testing
//
// 🎓 WHY: Test'lerde sık kullanılan objeler için helper yazmak
// test kodunu temiz ve okunabilir tutar.
func createTestWebsite(t *testing.T) *Website {
	t.Helper() // Go'ya bu bir helper olduğunu söyler

	tenantID := uuid.New()
	createdBy := uuid.New()

	website, err := New(
		tenantID,
		"Test Website",
		"test-website",
		TypeCMS,
		&createdBy,
	)

	require.NoError(t, err, "createTestWebsite should not return error")
	require.NotNil(t, website, "createTestWebsite should return website")

	return website
}

// ═════════════════════════════════════════════════════════════
// CONSTRUCTOR TESTS (New() fonksiyonu testleri)
// ═════════════════════════════════════════════════════════════

// TestNew_Success tests successful website creation
//
// 🎓 WHAT: Constructor'ın doğru çalıştığını test eder
// 🎓 WHY: New() en kritik fonksiyondur, doğru başlatma önemlidir
func TestNew_Success(t *testing.T) {
	// Arrange (Hazırlık)
	tenantID := uuid.New()
	createdBy := uuid.New()
	name := "My Awesome Website"
	slug := "my-awesome-website"
	websiteType := TypeBlog

	// Act (Aksiyon)
	website, err := New(tenantID, name, slug, websiteType, &createdBy)

	// Assert (Doğrulama)
	require.NoError(t, err, "New() should not return error")
	require.NotNil(t, website, "New() should return website")

	// Verify all fields are set correctly
	assert.NotEqual(t, uuid.Nil, website.ID, "ID should be generated")
	assert.Equal(t, tenantID, website.TenantID, "TenantID should match")
	assert.Equal(t, name, website.Name, "Name should match")
	assert.Equal(t, slug, website.Slug, "Slug should match")
	assert.Equal(t, websiteType, website.Type, "Type should match")
	assert.Equal(t, StatusDraft, website.Status, "Status should be draft")
	assert.Equal(t, name, website.Title, "Title should default to name")
	assert.Equal(t, slug+".keystone.dev", website.PrimaryURL, "PrimaryURL should be subdomain")
	assert.False(t, website.CustomDomainVerified, "CustomDomainVerified should be false")
	assert.Equal(t, &createdBy, website.CreatedBy, "CreatedBy should match")
	assert.NotZero(t, website.CreatedAt, "CreatedAt should be set")
	assert.NotZero(t, website.UpdatedAt, "UpdatedAt should be set")
}

// TestNew_ValidationErrors tests validation failures in constructor
//
// 🎓 TABLE-DRIVEN TEST PATTERN:
// Go'da birden fazla validation senaryosunu test etmenin en iyi yolu.
// Her test case bir struct olarak tanımlanır ve loop ile çalıştırılır.
func TestNew_ValidationErrors(t *testing.T) {
	tenantID := uuid.New()
	createdBy := uuid.New()

	// Test cases table
	// 🎓 WHY: Aynı test mantığını farklı inputlarla tekrarlamak yerine
	// tüm senaryoları bir tabloda tutarız.
	tests := []struct {
		name         string      // Test senaryosunun açıklaması
		tenantID     uuid.UUID   // Input: tenant ID
		websiteName  string      // Input: website name
		slug         string      // Input: slug
		websiteType  WebsiteType // Input: type
		expectedErr  error       // Beklenen hata
		errorMessage string      // Assert mesajı
	}{
		{
			name:         "Empty name should fail",
			tenantID:     tenantID,
			websiteName:  "", // EMPTY
			slug:         "test-slug",
			websiteType:  TypeCMS,
			expectedErr:  ErrNameRequired,
			errorMessage: "Empty name should return ErrNameRequired",
		},
		{
			name:         "Name too short should fail",
			tenantID:     tenantID,
			websiteName:  "A", // TOO SHORT (< 2 chars)
			slug:         "test-slug",
			websiteType:  TypeCMS,
			expectedErr:  ErrInvalidName,
			errorMessage: "Name with 1 char should return ErrInvalidName",
		},
		{
			name:         "Name too long should fail",
			tenantID:     tenantID,
			websiteName:  string(make([]byte, 101)), // TOO LONG (> 100 chars)
			slug:         "test-slug",
			websiteType:  TypeCMS,
			expectedErr:  ErrInvalidName,
			errorMessage: "Name with 101 chars should return ErrInvalidName",
		},
		{
			name:         "Empty slug should fail",
			tenantID:     tenantID,
			websiteName:  "Test Website",
			slug:         "", // EMPTY
			websiteType:  TypeCMS,
			expectedErr:  ErrSlugRequired,
			errorMessage: "Empty slug should return ErrSlugRequired",
		},
		{
			name:         "Slug too short should fail",
			tenantID:     tenantID,
			websiteName:  "Test Website",
			slug:         "a", // TOO SHORT (< 2 chars)
			websiteType:  TypeCMS,
			expectedErr:  ErrInvalidSlug,
			errorMessage: "Slug with 1 char should return ErrInvalidSlug",
		},
		{
			name:         "Slug too long should fail",
			tenantID:     tenantID,
			websiteName:  "Test Website",
			slug:         string(make([]byte, 51)), // TOO LONG (> 50 chars)
			websiteType:  TypeCMS,
			expectedErr:  ErrInvalidSlug,
			errorMessage: "Slug with 51 chars should return ErrInvalidSlug",
		},
		{
			name:         "Invalid website type should fail",
			tenantID:     tenantID,
			websiteName:  "Test Website",
			slug:         "test-slug",
			websiteType:  WebsiteType("invalid"), // INVALID TYPE
			expectedErr:  ErrInvalidWebsiteType,
			errorMessage: "Invalid type should return ErrInvalidWebsiteType",
		},
	}

	// Run all test cases
	// 🎓 t.Run(): Her case için alt-test oluşturur
	// Test output'ta "TestNew_ValidationErrors/Empty_name_should_fail" gibi görünür
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			website, err := New(tt.tenantID, tt.websiteName, tt.slug, tt.websiteType, &createdBy)

			// Assert
			assert.Error(t, err, tt.errorMessage)
			assert.ErrorIs(t, err, tt.expectedErr, "Error should be %v", tt.expectedErr)
			assert.Nil(t, website, "Website should be nil on validation error")
		})
	}
}

// ═════════════════════════════════════════════════════════════
// LIFECYCLE TESTS (Status değişimi testleri)
// ═════════════════════════════════════════════════════════════

// TestWebsite_Publish_Success tests successful publishing
//
// 🎓 LIFECYCLE: draft → published
func TestWebsite_Publish_Success(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	assert.Equal(t, StatusDraft, website.Status, "Initial status should be draft")

	// Act
	err := website.Publish()

	// Assert
	require.NoError(t, err, "Publish should not return error")
	assert.Equal(t, StatusPublished, website.Status, "Status should be published")
	assert.NotNil(t, website.PublishedAt, "PublishedAt should be set")
	assert.True(t, website.IsPublished(), "IsPublished() should return true")
}

// TestWebsite_Publish_AlreadyPublished tests error when already published
//
// 🎓 EDGE CASE: İkinci kez publish etmeye çalışma
func TestWebsite_Publish_AlreadyPublished(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.Publish()
	require.NoError(t, err, "First publish should succeed")

	// Act
	err = website.Publish()

	// Assert
	assert.Error(t, err, "Second publish should fail")
	assert.ErrorIs(t, err, ErrAlreadyPublished, "Error should be ErrAlreadyPublished")
}

// TestWebsite_Publish_ArchivedWebsite tests error when publishing archived website
//
// 🎓 BUSINESS RULE: Arşivlenmiş website publish edilemez
func TestWebsite_Publish_ArchivedWebsite(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.Archive()
	require.NoError(t, err, "Archive should succeed")

	// Act
	err = website.Publish()

	// Assert
	assert.Error(t, err, "Publishing archived website should fail")
	assert.ErrorIs(t, err, ErrCannotPublishArchived, "Error should be ErrCannotPublishArchived")
}

// TestWebsite_Unpublish_Success tests successful unpublishing
//
// 🎓 LIFECYCLE: published → unpublished
func TestWebsite_Unpublish_Success(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.Publish()
	require.NoError(t, err, "Publish should succeed")

	// Act
	err = website.Unpublish()

	// Assert
	require.NoError(t, err, "Unpublish should not return error")
	assert.Equal(t, StatusUnpublished, website.Status, "Status should be unpublished")
	assert.NotNil(t, website.UnpublishedAt, "UnpublishedAt should be set")
	assert.False(t, website.IsPublished(), "IsPublished() should return false")
}

// TestWebsite_Unpublish_NotPublished tests error when unpublishing non-published website
//
// 🎓 BUSINESS RULE: Sadece published website unpublish edilebilir
func TestWebsite_Unpublish_NotPublished(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	assert.Equal(t, StatusDraft, website.Status, "Initial status should be draft")

	// Act
	err := website.Unpublish()

	// Assert
	assert.Error(t, err, "Unpublishing draft should fail")
	assert.ErrorIs(t, err, ErrNotPublished, "Error should be ErrNotPublished")
}

// TestWebsite_Archive_Success tests successful archiving
//
// 🎓 LIFECYCLE: any status → archived
func TestWebsite_Archive_Success(t *testing.T) {
	// Test archiving from different statuses
	tests := []struct {
		name          string
		initialStatus Status
		setupFunc     func(*Website) error
	}{
		{
			name:          "Archive from draft",
			initialStatus: StatusDraft,
			setupFunc:     func(w *Website) error { return nil },
		},
		{
			name:          "Archive from published",
			initialStatus: StatusPublished,
			setupFunc:     func(w *Website) error { return w.Publish() },
		},
		{
			name:          "Archive from unpublished",
			initialStatus: StatusUnpublished,
			setupFunc: func(w *Website) error {
				if err := w.Publish(); err != nil {
					return err
				}
				return w.Unpublish()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			website := createTestWebsite(t)
			err := tt.setupFunc(website)
			require.NoError(t, err, "Setup should succeed")
			assert.Equal(t, tt.initialStatus, website.Status, "Initial status should be %s", tt.initialStatus)

			// Act
			err = website.Archive()

			// Assert
			require.NoError(t, err, "Archive should not return error")
			assert.Equal(t, StatusArchived, website.Status, "Status should be archived")
			assert.NotNil(t, website.ArchivedAt, "ArchivedAt should be set")
			assert.True(t, website.IsArchived(), "IsArchived() should return true")
		})
	}
}

// TestWebsite_Archive_AlreadyArchived tests error when already archived
func TestWebsite_Archive_AlreadyArchived(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.Archive()
	require.NoError(t, err, "First archive should succeed")

	// Act
	err = website.Archive()

	// Assert
	assert.Error(t, err, "Second archive should fail")
	assert.ErrorIs(t, err, ErrAlreadyArchived, "Error should be ErrAlreadyArchived")
}

// TestWebsite_Restore_Success tests successful restoration from archive
//
// 🎓 LIFECYCLE: archived → draft
func TestWebsite_Restore_Success(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.Archive()
	require.NoError(t, err, "Archive should succeed")

	// Act
	err = website.Restore()

	// Assert
	require.NoError(t, err, "Restore should not return error")
	assert.Equal(t, StatusDraft, website.Status, "Status should be draft")
	assert.Nil(t, website.ArchivedAt, "ArchivedAt should be nil after restore")
	assert.True(t, website.IsDraft(), "IsDraft() should return true")
}

// TestWebsite_Restore_NotArchived tests error when restoring non-archived website
func TestWebsite_Restore_NotArchived(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	assert.Equal(t, StatusDraft, website.Status, "Initial status should be draft")

	// Act
	err := website.Restore()

	// Assert
	assert.Error(t, err, "Restoring non-archived should fail")
	assert.ErrorIs(t, err, ErrNotArchived, "Error should be ErrNotArchived")
}

// ═════════════════════════════════════════════════════════════
// CUSTOM DOMAIN TESTS (Özel alan adı testleri)
// ═════════════════════════════════════════════════════════════

// TestWebsite_SetCustomDomain_Success tests setting custom domain
//
// 🎓 DOMAIN MANAGEMENT: Özel alan adı ekleme
func TestWebsite_SetCustomDomain_Success(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	customDomain := "example.com"

	// Act
	err := website.SetCustomDomain(customDomain)

	// Assert
	require.NoError(t, err, "SetCustomDomain should not return error")
	assert.NotNil(t, website.CustomDomain, "CustomDomain should be set")
	assert.Equal(t, customDomain, *website.CustomDomain, "CustomDomain should match")
	assert.False(t, website.CustomDomainVerified, "CustomDomainVerified should be false initially")
	assert.True(t, website.HasCustomDomain(), "HasCustomDomain() should return true")
	assert.False(t, website.IsCustomDomainVerified(), "IsCustomDomainVerified() should return false")
}

// TestWebsite_SetCustomDomain_EmptyDomain tests error with empty domain
func TestWebsite_SetCustomDomain_EmptyDomain(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)

	// Act
	err := website.SetCustomDomain("")

	// Assert
	assert.Error(t, err, "SetCustomDomain with empty string should fail")
	assert.ErrorIs(t, err, ErrInvalidCustomDomain, "Error should be ErrInvalidCustomDomain")
}

// TestWebsite_VerifyCustomDomain_Success tests domain verification
//
// 🎓 DOMAIN VERIFICATION: DNS doğrulaması sonrası
func TestWebsite_VerifyCustomDomain_Success(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	customDomain := "example.com"
	err := website.SetCustomDomain(customDomain)
	require.NoError(t, err, "SetCustomDomain should succeed")

	// Act
	err = website.VerifyCustomDomain()

	// Assert
	require.NoError(t, err, "VerifyCustomDomain should not return error")
	assert.True(t, website.CustomDomainVerified, "CustomDomainVerified should be true")
	assert.Equal(t, customDomain, website.PrimaryURL, "PrimaryURL should be custom domain")
	assert.True(t, website.IsCustomDomainVerified(), "IsCustomDomainVerified() should return true")
}

// TestWebsite_VerifyCustomDomain_NotSet tests error when domain not set
func TestWebsite_VerifyCustomDomain_NotSet(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	assert.Nil(t, website.CustomDomain, "CustomDomain should be nil")

	// Act
	err := website.VerifyCustomDomain()

	// Assert
	assert.Error(t, err, "VerifyCustomDomain without domain should fail")
	assert.ErrorIs(t, err, ErrCustomDomainNotSet, "Error should be ErrCustomDomainNotSet")
}

// TestWebsite_VerifyCustomDomain_AlreadyVerified tests error when already verified
func TestWebsite_VerifyCustomDomain_AlreadyVerified(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.SetCustomDomain("example.com")
	require.NoError(t, err, "SetCustomDomain should succeed")
	err = website.VerifyCustomDomain()
	require.NoError(t, err, "First verification should succeed")

	// Act
	err = website.VerifyCustomDomain()

	// Assert
	assert.Error(t, err, "Second verification should fail")
	assert.ErrorIs(t, err, ErrCustomDomainAlreadyVerified, "Error should be ErrCustomDomainAlreadyVerified")
}

// TestWebsite_RemoveCustomDomain tests domain removal
//
// 🎓 CLEANUP: Özel alan adını kaldırma ve subdomain'e geri dönüş
func TestWebsite_RemoveCustomDomain(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	err := website.SetCustomDomain("example.com")
	require.NoError(t, err, "SetCustomDomain should succeed")
	err = website.VerifyCustomDomain()
	require.NoError(t, err, "VerifyCustomDomain should succeed")

	originalSubdomain := website.Slug + ".keystone.dev"

	// Act
	website.RemoveCustomDomain()

	// Assert
	assert.Nil(t, website.CustomDomain, "CustomDomain should be nil")
	assert.False(t, website.CustomDomainVerified, "CustomDomainVerified should be false")
	assert.Equal(t, originalSubdomain, website.PrimaryURL, "PrimaryURL should revert to subdomain")
	assert.False(t, website.HasCustomDomain(), "HasCustomDomain() should return false")
}

// ═════════════════════════════════════════════════════════════
// TEMPLATE TESTS (Şablon yönetimi testleri)
// ═════════════════════════════════════════════════════════════

// TestWebsite_SetTemplate tests template association
//
// 🎓 TEMPLATE SYSTEM: Website'a şablon atama
func TestWebsite_SetTemplate(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	templateID := uuid.New()
	templateName := "Modern Business Template"
	templateVersion := "v1.0.0"

	// Act
	website.SetTemplate(templateID, templateName, templateVersion)

	// Assert
	assert.NotNil(t, website.TemplateID, "TemplateID should be set")
	assert.Equal(t, templateID, *website.TemplateID, "TemplateID should match")
	assert.NotNil(t, website.TemplateName, "TemplateName should be set")
	assert.Equal(t, templateName, *website.TemplateName, "TemplateName should match")
	assert.NotNil(t, website.TemplateVersion, "TemplateVersion should be set")
	assert.Equal(t, templateVersion, *website.TemplateVersion, "TemplateVersion should match")
}

// ═════════════════════════════════════════════════════════════
// UPDATE TESTS (Güncelleme testleri)
// ═════════════════════════════════════════════════════════════

// TestWebsite_Update tests basic update
func TestWebsite_Update(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	originalUpdatedAt := website.UpdatedAt
	time.Sleep(1 * time.Millisecond) // Ensure time difference

	newName := "Updated Website Name"
	newDescription := "Updated description"

	// Act
	website.Update(&newName, &newDescription)

	// Assert
	assert.Equal(t, newName, website.Name, "Name should be updated")
	assert.NotNil(t, website.Description, "Description should be set")
	assert.Equal(t, newDescription, *website.Description, "Description should match")
	assert.True(t, website.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be updated")
}

// TestWebsite_UpdateBasicInfo tests basic info update
func TestWebsite_UpdateBasicInfo(t *testing.T) {
	// Arrange
	website := createTestWebsite(t)
	originalUpdatedAt := website.UpdatedAt
	time.Sleep(1 * time.Millisecond)

	newName := "New Name"
	newTitle := "New Title"
	newDescription := "New Description"

	// Act
	website.UpdateBasicInfo(newName, newTitle, &newDescription)

	// Assert
	assert.Equal(t, newName, website.Name, "Name should be updated")
	assert.Equal(t, newTitle, website.Title, "Title should be updated")
	assert.NotNil(t, website.Description, "Description should be set")
	assert.Equal(t, newDescription, *website.Description, "Description should match")
	assert.True(t, website.UpdatedAt.After(originalUpdatedAt), "UpdatedAt should be updated")
}

// ═════════════════════════════════════════════════════════════
// STATUS & TYPE VALIDATION TESTS
// ═════════════════════════════════════════════════════════════

// TestStatus_IsValid tests status validation
func TestStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		status   Status
		expected bool
	}{
		{"Draft is valid", StatusDraft, true},
		{"Published is valid", StatusPublished, true},
		{"Archived is valid", StatusArchived, true},
		{"Unpublished is valid", StatusUnpublished, true},
		{"Invalid status", Status("invalid"), false},
		{"Empty status", Status(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.IsValid()
			assert.Equal(t, tt.expected, result, "IsValid() should return %v for %s", tt.expected, tt.status)
		})
	}
}

// TestWebsiteType_IsValid tests type validation
func TestWebsiteType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		wType    WebsiteType
		expected bool
	}{
		{"CMS is valid", TypeCMS, true},
		{"CRM is valid", TypeCRM, true},
		{"Ecommerce is valid", TypeEcommerce, true},
		{"Education is valid", TypeEducation, true},
		{"Hospitality is valid", TypeHospitality, true},
		{"ERP is valid", TypeERP, true},
		{"Blog is valid", TypeBlog, true},
		{"Portfolio is valid", TypePortfolio, true},
		{"Custom is valid", TypeCustom, true},
		{"Invalid type", WebsiteType("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.wType.IsValid()
			assert.Equal(t, tt.expected, result, "IsValid() should return %v for %s", tt.expected, tt.wType)
		})
	}
}

// TestWebsiteType_GetDescription tests type descriptions
//
// 🎓 ENUM WITH DESCRIPTIONS: Her enum değeri için human-readable açıklama
func TestWebsiteType_GetDescription(t *testing.T) {
	tests := []struct {
		wType       WebsiteType
		description string
	}{
		{TypeCMS, "Content Management System"},
		{TypeCRM, "Customer Relationship Management"},
		{TypeEcommerce, "E-commerce Platform"},
		{TypeEducation, "Educational Platform"},
		{TypeHospitality, "Hospitality Management"},
		{TypeERP, "Enterprise Resource Planning"},
		{TypeBlog, "Blog Platform"},
		{TypePortfolio, "Portfolio Website"},
		{TypeCustom, "Custom Website"},
		{WebsiteType("invalid"), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(string(tt.wType), func(t *testing.T) {
			result := tt.wType.GetDescription()
			assert.Equal(t, tt.description, result, "GetDescription() should return correct description")
		})
	}
}

// ═════════════════════════════════════════════════════════════
// HELPER METHOD TESTS (Yardımcı metodlar)
// ═════════════════════════════════════════════════════════════

// TestWebsite_CanPublish tests publish eligibility
//
// 🎓 BUSINESS RULE: Website ancak draft durumunda VE homepage varsa publish edilebilir
func TestWebsite_CanPublish(t *testing.T) {
	tests := []struct {
		name       string
		status     Status
		homepageID *uuid.UUID
		expected   bool
	}{
		{
			name:       "Draft with homepage can publish",
			status:     StatusDraft,
			homepageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			expected:   true,
		},
		{
			name:       "Draft without homepage cannot publish",
			status:     StatusDraft,
			homepageID: nil,
			expected:   false,
		},
		{
			name:       "Published cannot publish again",
			status:     StatusPublished,
			homepageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			expected:   false,
		},
		{
			name:       "Archived cannot publish",
			status:     StatusArchived,
			homepageID: func() *uuid.UUID { id := uuid.New(); return &id }(),
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			website := createTestWebsite(t)
			website.Status = tt.status
			website.HomepageID = tt.homepageID

			// Act
			result := website.CanPublish()

			// Assert
			assert.Equal(t, tt.expected, result, "CanPublish() should return %v", tt.expected)
		})
	}
}

// 🎓 TEST COVERAGE SUMMARY:
//
// ✅ Constructor (New)
//    - Success case
//    - 7 validation error cases
//
// ✅ Lifecycle Methods
//    - Publish (success, already published, archived)
//    - Unpublish (success, not published)
//    - Archive (from 3 different statuses, already archived)
//    - Restore (success, not archived)
//
// ✅ Custom Domain Management
//    - SetCustomDomain (success, empty domain)
//    - VerifyCustomDomain (success, not set, already verified)
//    - RemoveCustomDomain
//
// ✅ Template Management
//    - SetTemplate
//
// ✅ Update Methods
//    - Update (basic)
//    - UpdateBasicInfo
//
// ✅ Validation
//    - Status.IsValid (6 cases)
//    - WebsiteType.IsValid (10 cases)
//    - WebsiteType.GetDescription (10 cases)
//
// ✅ Helper Methods
//    - CanPublish (4 cases)
//    - IsDraft, IsPublished, IsArchived
//    - HasCustomDomain, IsCustomDomainVerified
//
// Total Test Cases: 40+
// Expected Coverage: ~85-90%
