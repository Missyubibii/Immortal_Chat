# Unit Tests for Dispatcher Service

## Overview

Comprehensive unit tests for the `dispatcher.go` service using **testify/mock** to test webhook processing logic without requiring a real database or Redis instance.

## Test Coverage

**Coverage: 67.1% of statements**

### Test Cases

1. **TestProcessWebhook_ValidUserMessage**

   - ✅ Tests successful processing of a valid user message
   - Verifies message is saved to database
   - Confirms conversation is created
   - Checks deduplication marking

2. **TestProcessWebhook_EchoMessage**

   - ✅ Tests that echo messages (sent by page) are filtered out
   - Verifies no message saved to database
   - Per user requirement: Only user messages should be processed

3. **TestProcessWebhook_DuplicateMessage**

   - ✅ Tests deduplication logic
   - Confirms duplicate messages are skipped
   - Verifies idempotency (same webhook can be received multiple times safely)

4. **TestProcessWebhook_InvalidJSON**

   - ✅ Tests handling of malformed JSON payloads
   - Ensures webhook status is set to "failed"
   - Confirms no panic occurs

5. **TestProcessWebhook_DedupError**

   - ✅ Tests error handling when Redis dedup check fails
   - Verifies graceful degradation
   - Ensures message is not saved if dedup check errors

6. **TestProcessWebhook_SaveMessageError**

   - ✅ Tests error handling when database save fails
   - Confirms error is logged but doesn't crash
   - Verifies panic recovery works

7. **TestProcessWebhook_PanicRecovery**
   - ✅ Tests that panics are recovered gracefully
   - Critical per user requirement: Must not crash Docker container
   - Simulates panic in dedup check

## Running Tests

### Run All Tests

```bash
go test ./internal/core/services/ -v
```

### Run Specific Test

```bash
go test ./internal/core/services/ -v -run TestProcessWebhook_ValidUserMessage
```

### Run with Coverage

```bash
go test ./internal/core/services/ -v -cover
```

### Generate Coverage Report

```bash
go test ./internal/core/services/ -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Mock Repositories

The tests use mock implementations of all repository interfaces:

- `MockWebhookRepository` - Mocks webhook log persistence
- `MockMessageRepository` - Mocks message persistence
- `MockConversationRepository` - Mocks conversation management
- `MockDedupRepository` - Mocks Redis deduplication

**Benefit**: Tests run in milliseconds without database dependencies

## Test Data Helpers

### `createValidUserMessagePayload()`

Creates a realistic Facebook webhook JSON with a user text message

### `createEchoMessagePayload()`

Creates echo message payload (is_echo = true) for filter testing

## Example Test Output

```
=== RUN   TestProcessWebhook_ValidUserMessage
--- PASS: TestProcessWebhook_ValidUserMessage (0.20s)
=== RUN   TestProcessWebhook_EchoMessage
--- PASS: TestProcessWebhook_EchoMessage (0.20s)
=== RUN   TestProcessWebhook_DuplicateMessage
--- PASS: TestProcessWebhook_DuplicateMessage (0.20s)
=== RUN   TestProcessWebhook_InvalidJSON
--- PASS: TestProcessWebhook_InvalidJSON (0.20s)
=== RUN   TestProcessWebhook_DedupError
--- PASS: TestProcessWebhook_DedupError (0.20s)
=== RUN   TestProcessWebhook_SaveMessageError
--- PASS: TestProcessWebhook_SaveMessageError (0.20s)
=== RUN   TestProcessWebhook_PanicRecovery
--- PASS: TestProcessWebhook_PanicRecovery (0.20s)
PASS
coverage: 67.1% of statements
ok      immortal-chat/internal/core/services    1.946s
```

## Key Features Tested

✅ **Message Filtering** - Echo, delivery, read receipts excluded  
✅ **Deduplication** - Duplicate webhooks handled correctly  
✅ **Error Handling** - All repository errors handled gracefully  
✅ **Panic Recovery** - Panics caught and logged, no container crash  
✅ **Async Processing** - Tests account for goroutine timing  
✅ **Status Lifecycle** - Webhook status: pending → processed/failed

## Dependencies

```bash
go get github.com/stretchr/testify
```

**Packages used**:

- `github.com/stretchr/testify/assert` - Assertions
- `github.com/stretchr/testify/mock` - Mock objects

## Integration with CI/CD

Add to your CI pipeline:

```yaml
- name: Run Unit Tests
  run: go test ./... -v -cover

- name: Generate Coverage
  run: go test ./... -coverprofile=coverage.out
```

## Future Test Additions

Potential additional test cases:

- [ ] Multiple messages in single webhook
- [ ] Image/attachment message processing
- [ ] Conversation lookup vs creation
- [ ] Concurrent webhook processing
- [ ] Webhook retry logic
