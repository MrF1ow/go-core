package log

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// resetServiceInstance tears down the singleton so each test starts fresh.
func resetServiceInstance() {
	if serviceInstance != nil {
		serviceInstance.cancel()
		time.Sleep(15 * time.Millisecond)
		serviceInstance = nil
	}
}

// initTestService creates a log service with a nil pool (no DB writes),
// then cancels the context to stop the background worker so entries
// remain in the channel for inspection.
func initTestService() *Service {
	resetServiceInstance()
	svc := InitializeLogService(nil, nil)
	svc.cancel()
	time.Sleep(15 * time.Millisecond)
	return svc
}

// drainChannel reads all pending entries from the log channel.
func drainChannel(s *Service) []LogEntry {
	var entries []LogEntry
	for {
		select {
		case e := <-s.logChannel:
			entries = append(entries, e)
		default:
			return entries
		}
	}
}

func TestLogActivity_SkipsNilUserID(t *testing.T) {
	svc := initTestService()
	defer resetServiceInstance()

	appID := uuid.New()
	svc.LogActivity(appID, uuid.Nil, EventLoginFailed, "1.2.3.4", "test-agent", map[string]interface{}{
		"email": "unknown@example.com",
	})

	entries := drainChannel(svc)
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries for nil userID, got %d", len(entries))
	}
}

func TestLogActivity_QueuesValidUserID(t *testing.T) {
	svc := initTestService()
	defer resetServiceInstance()

	appID := uuid.New()
	userID := uuid.New()
	svc.LogActivity(appID, userID, EventLogin, "1.2.3.4", "test-agent", map[string]interface{}{
		"method": "password",
	})

	entries := drainChannel(svc)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry for valid userID, got %d", len(entries))
	}

	if entries[0].UserID != userID {
		t.Fatalf("Expected userID %s, got %s", userID, entries[0].UserID)
	}
	if entries[0].EventType != EventLogin {
		t.Fatalf("Expected event %s, got %s", EventLogin, entries[0].EventType)
	}
}

func TestLogActivityWithAnomalyResult_SkipsNilUserID(t *testing.T) {
	svc := initTestService()
	defer resetServiceInstance()

	appID := uuid.New()
	result := &AnomalyResult{
		IsAnomaly: true,
		Severity:  "high",
		Reasons:   []string{"new_ip"},
	}

	svc.LogActivityWithAnomalyResult(appID, uuid.Nil, "test@example.com", EventBruteForceDetected, "1.2.3.4", "test-agent", nil, result)

	entries := drainChannel(svc)
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries for nil userID with anomaly result, got %d", len(entries))
	}
}

func TestLogLoginFailed_UsesProvidedUserID(t *testing.T) {
	svc := initTestService()
	defer resetServiceInstance()

	_ = svc
	appID := uuid.New()
	userID := uuid.New()

	LogLoginFailed(appID, userID, "1.2.3.4", "test-agent", map[string]interface{}{
		"email": "user@example.com",
	})

	entries := drainChannel(GetLogService())
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry for valid userID, got %d", len(entries))
	}
	if entries[0].UserID != userID {
		t.Fatalf("Expected userID %s, got %s", userID, entries[0].UserID)
	}
	if entries[0].EventType != EventLoginFailed {
		t.Fatalf("Expected event %s, got %s", EventLoginFailed, entries[0].EventType)
	}
}

func TestLogLoginFailed_SkipsNilUserID(t *testing.T) {
	svc := initTestService()
	defer resetServiceInstance()

	_ = svc
	appID := uuid.New()

	LogLoginFailed(appID, uuid.Nil, "1.2.3.4", "test-agent", map[string]interface{}{
		"email": "nonexistent@example.com",
	})

	entries := drainChannel(GetLogService())
	if len(entries) != 0 {
		t.Fatalf("Expected 0 entries for nil userID, got %d", len(entries))
	}
}
