package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"immortal-chat/internal/core/domain"
)

// ============================================================================
// Mock Repositories
// ============================================================================

// MockWebhookRepository mocks WebhookRepository interface
type MockWebhookRepository struct {
	mock.Mock
}

func (m *MockWebhookRepository) SaveLog(ctx context.Context, log *domain.WebhookLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockWebhookRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// MockMessageRepository mocks MessageRepository interface
type MockMessageRepository struct {
	mock.Mock
}

func (m *MockMessageRepository) SaveMessage(ctx context.Context, msg *domain.Message) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockMessageRepository) GetByID(ctx context.Context, id string) (*domain.Message, error) {
	args := m.Called(ctx, id)
	// Safely handle nil return
	if result := args.Get(0); result != nil {
		return result.(*domain.Message), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockMessageRepository) Exists(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// MockConversationRepository mocks ConversationRepository interface
type MockConversationRepository struct {
	mock.Mock
}

func (m *MockConversationRepository) GetOrCreate(ctx context.Context, platform, externalID string) (int64, error) {
	args := m.Called(ctx, platform, externalID)
	// Safely get int64
	return int64(args.Int(0)), args.Error(1)
}

// MockDedupRepository mocks DedupRepository interface
type MockDedupRepository struct {
	mock.Mock
}

func (m *MockDedupRepository) IsDuplicate(ctx context.Context, eventID string) (bool, error) {
	args := m.Called(ctx, eventID)
	return args.Bool(0), args.Error(1)
}

func (m *MockDedupRepository) MarkProcessed(ctx context.Context, eventID string, ttl time.Duration) error {
	args := m.Called(ctx, eventID, ttl)
	return args.Error(0)
}

// ============================================================================
// Test Helper Functions
// ============================================================================

// createTestDispatcher creates a dispatcher with mock repositories
func createTestDispatcher() (*Dispatcher, *MockWebhookRepository, *MockMessageRepository, *MockConversationRepository, *MockDedupRepository) {
	webhookRepo := new(MockWebhookRepository)
	messageRepo := new(MockMessageRepository)
	conversationRepo := new(MockConversationRepository)
	dedupRepo := new(MockDedupRepository)

	dispatcher := NewDispatcher(webhookRepo, messageRepo, conversationRepo, dedupRepo)

	return dispatcher, webhookRepo, messageRepo, conversationRepo, dedupRepo
}

// createValidUserMessagePayload creates a valid Facebook webhook JSON with a user message
func createValidUserMessagePayload() []byte {
	payload := map[string]interface{}{
		"object": "page",
		"entry": []map[string]interface{}{
			{
				"id":   "123456",
				"time": 1234567890,
				"messaging": []map[string]interface{}{
					{
						"sender": map[string]string{
							"id": "USER_PSID_123",
						},
						"recipient": map[string]string{
							"id": "PAGE_ID_456",
						},
						"timestamp": 1234567890,
						"message": map[string]interface{}{
							"mid":  "mid.test123",
							"text": "Hello, this is a test message",
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(payload)
	return data
}

// createEchoMessagePayload creates a webhook with echo message (should be filtered)
func createEchoMessagePayload() []byte {
	payload := map[string]interface{}{
		"object": "page",
		"entry": []map[string]interface{}{
			{
				"id":   "123456",
				"time": 1234567890,
				"messaging": []map[string]interface{}{
					{
						"sender": map[string]string{
							"id": "PAGE_ID_456",
						},
						"recipient": map[string]string{
							"id": "USER_PSID_123",
						},
						"timestamp": 1234567890,
						"message": map[string]interface{}{
							"mid":     "mid.echo123",
							"text":    "This is an echo",
							"is_echo": true, // CRITICAL: Echo message
						},
					},
				},
			},
		},
	}

	data, _ := json.Marshal(payload)
	return data
}

// ============================================================================
// Unit Tests
// ============================================================================

// TestProcessWebhook_ValidUserMessage tests processing a valid user message
func TestProcessWebhook_ValidUserMessage(t *testing.T) {
	dispatcher, webhookRepo, messageRepo, conversationRepo, dedupRepo := createTestDispatcher()

	payload := createValidUserMessagePayload()
	ctx := context.Background()

	// Setup expectations
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	webhookRepo.On("UpdateStatus", mock.Anything, mock.Anything, domain.WebhookStatusProcessed).Return(nil).Maybe()
	dedupRepo.On("IsDuplicate", ctx, "mid.test123").Return(false, nil)
	conversationRepo.On("GetOrCreate", ctx, "facebook", "USER_PSID_123").Return(1, nil)
	messageRepo.On("SaveMessage", ctx, mock.MatchedBy(func(msg *domain.Message) bool {
		return msg.ID == "mid.test123" &&
			msg.Content == "Hello, this is a test message" &&
			msg.SenderID == "USER_PSID_123"
	})).Return(nil)
	dedupRepo.On("MarkProcessed", ctx, "mid.test123", 24*time.Hour).Return(nil)

	// Execute
	dispatcher.ProcessWebhook(ctx, "facebook", payload)

	// Give async goroutines time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify
	messageRepo.AssertExpectations(t)
	conversationRepo.AssertExpectations(t)
	dedupRepo.AssertCalled(t, "IsDuplicate", ctx, "mid.test123")
}

// TestProcessWebhook_EchoMessage tests that echo messages are filtered out
func TestProcessWebhook_EchoMessage(t *testing.T) {
	dispatcher, webhookRepo, messageRepo, conversationRepo, dedupRepo := createTestDispatcher()

	payload := createEchoMessagePayload()
	ctx := context.Background()

	// Setup expectations - only webhook log should be saved, no message processing
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	webhookRepo.On("UpdateStatus", mock.Anything, mock.Anything, domain.WebhookStatusProcessed).Return(nil).Maybe()

	// Execute
	dispatcher.ProcessWebhook(ctx, "facebook", payload)

	// Give async goroutines time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify no message processing occurred
	messageRepo.AssertNotCalled(t, "SaveMessage", mock.Anything, mock.Anything)
	conversationRepo.AssertNotCalled(t, "GetOrCreate", mock.Anything, mock.Anything, mock.Anything)
	dedupRepo.AssertNotCalled(t, "IsDuplicate", mock.Anything, mock.Anything)
}

// TestProcessWebhook_DuplicateMessage tests deduplication logic
func TestProcessWebhook_DuplicateMessage(t *testing.T) {
	dispatcher, webhookRepo, messageRepo, conversationRepo, dedupRepo := createTestDispatcher()

	payload := createValidUserMessagePayload()
	ctx := context.Background()

	// Setup expectations
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	webhookRepo.On("UpdateStatus", mock.Anything, mock.Anything, domain.WebhookStatusProcessed).Return(nil).Maybe()
	dedupRepo.On("IsDuplicate", ctx, "mid.test123").Return(true, nil) // Mark as duplicate

	// Execute
	dispatcher.ProcessWebhook(ctx, "facebook", payload)

	// Give async goroutines time to complete
	time.Sleep(200 * time.Millisecond)

	// Verify - dedup check was called, but message was NOT saved
	dedupRepo.AssertExpectations(t)
	messageRepo.AssertNotCalled(t, "SaveMessage", mock.Anything, mock.Anything)
	conversationRepo.AssertNotCalled(t, "GetOrCreate", mock.Anything, mock.Anything, mock.Anything)
}

// TestProcessWebhook_InvalidJSON tests handling of malformed JSON
func TestProcessWebhook_InvalidJSON(t *testing.T) {
	dispatcher, webhookRepo, _, _, _ := createTestDispatcher()

	invalidPayload := []byte(`{"invalid json`)
	ctx := context.Background()

	// Setup expectations
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	webhookRepo.On("UpdateStatus", mock.Anything, mock.Anything, domain.WebhookStatusFailed).Return(nil).Maybe()

	// Execute - should not panic
	assert.NotPanics(t, func() {
		dispatcher.ProcessWebhook(ctx, "facebook", invalidPayload)
		time.Sleep(200 * time.Millisecond)
	})
}

// TestProcessWebhook_DedupError tests handling of dedup check errors
func TestProcessWebhook_DedupError(t *testing.T) {
	dispatcher, webhookRepo, messageRepo, _, dedupRepo := createTestDispatcher()

	payload := createValidUserMessagePayload()
	ctx := context.Background()

	// Setup expectations - dedup check returns error
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	webhookRepo.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	dedupRepo.On("IsDuplicate", ctx, "mid.test123").Return(false, errors.New("redis connection error"))

	// Execute - should handle error gracefully
	assert.NotPanics(t, func() {
		dispatcher.ProcessWebhook(ctx, "facebook", payload)
		time.Sleep(200 * time.Millisecond)
	})

	// Message should not be saved due to error
	messageRepo.AssertNotCalled(t, "SaveMessage", mock.Anything, mock.Anything)
}

// TestProcessWebhook_SaveMessageError tests handling of save message errors
func TestProcessWebhook_SaveMessageError(t *testing.T) {
	dispatcher, webhookRepo, messageRepo, conversationRepo, dedupRepo := createTestDispatcher()

	payload := createValidUserMessagePayload()
	ctx := context.Background()

	// Setup expectations
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	webhookRepo.On("UpdateStatus", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	dedupRepo.On("IsDuplicate", ctx, "mid.test123").Return(false, nil)
	conversationRepo.On("GetOrCreate", ctx, "facebook", "USER_PSID_123").Return(1, nil)
	messageRepo.On("SaveMessage", ctx, mock.Anything).Return(errors.New("database error"))

	// Execute - should handle error gracefully
	assert.NotPanics(t, func() {
		dispatcher.ProcessWebhook(ctx, "facebook", payload)
		time.Sleep(200 * time.Millisecond)
	})

	// Verify save was attempted
	messageRepo.AssertExpectations(t)
}

// TestProcessWebhook_PanicRecovery tests that panics are recovered gracefully
func TestProcessWebhook_PanicRecovery(t *testing.T) {
	dispatcher, webhookRepo, _, _, dedupRepo := createTestDispatcher()

	payload := createValidUserMessagePayload()
	ctx := context.Background()

	// Setup expectations - make dedup check panic
	webhookRepo.On("SaveLog", ctx, mock.AnythingOfType("*domain.WebhookLog")).Return(nil)
	dedupRepo.On("IsDuplicate", ctx, "mid.test123").Run(func(args mock.Arguments) {
		panic("simulated panic in dedup check")
	}).Return(false, nil)

	// Execute - should recover from panic and not crash
	assert.NotPanics(t, func() {
		dispatcher.ProcessWebhook(ctx, "facebook", payload)
		time.Sleep(200 * time.Millisecond)
	})
}
