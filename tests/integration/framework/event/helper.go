package event

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/registration"
)

type Helper struct {
	pool *pgxpool.Pool
}

func NewHelper(pool *pgxpool.Pool) *Helper {
	return &Helper{pool: pool}
}

// WaitForEvent waits for an event to appear in the database
func (h *Helper) WaitForEvent(t *testing.T, eventType string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for event %s", eventType)
		case <-ticker.C:
			if h.eventExists(eventType) {
				return
			}
		}
	}
}

func (h *Helper) eventExists(eventType string) bool {
	var count int
	query := fmt.Sprintf(`
        SELECT COUNT(*) FROM watermill_events_registration 
        WHERE metadata->>'name' = $1
    `)

	_ = h.pool.QueryRow(context.Background(), query, eventType).Scan(&count)
	return count > 0
}

// AssertEvent retrieves and asserts on a specific event
func (h *Helper) AssertEvent(t *testing.T, eventType string) *EventAssertion {
	t.Helper()

	h.WaitForEvent(t, eventType, 5*time.Second)

	var payload json.RawMessage
	var metadata json.RawMessage
	var offset int64

	query := fmt.Sprintf(`
        SELECT payload, metadata, "offset"
        FROM watermill_events_registration
        WHERE metadata->>'name' = $1
        ORDER BY "offset" DESC
        LIMIT 1
    `)

	err := h.pool.QueryRow(context.Background(), query, eventType).Scan(&payload, &metadata, &offset)
	require.NoError(t, err, "event %s not found", eventType)

	return &EventAssertion{
		t:         t,
		eventType: eventType,
		payload:   payload,
		metadata:  metadata,
		offset:    offset,
	}
}

// AssertNoEvent ensures no event of the given type exists
func (h *Helper) AssertNoEvent(t *testing.T, eventType string) {
	t.Helper()

	var count int
	query := fmt.Sprintf(`
        SELECT COUNT(*) FROM watermill_events_registration 
        WHERE metadata->>'name' = $1
    `)

	err := h.pool.QueryRow(context.Background(), query, eventType).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "expected no %s events, but found %d", eventType, count)
}

// AssertEventCount verifies the number of events of a specific type
func (h *Helper) AssertEventCount(t *testing.T, eventType string, expected int) {
	t.Helper()

	var count int
	query := fmt.Sprintf(`
        SELECT COUNT(*) FROM watermill_events_registration 
        WHERE metadata->>'name' = $1
    `)

	err := h.pool.QueryRow(context.Background(), query, eventType).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, expected, count, "unexpected %s event count", eventType)
}

// AssertRegistrationStartedEvent is a convenience method
func (h *Helper) AssertRegistrationStartedEvent(t *testing.T, email string) *RegistrationStartedAssertion {
	t.Helper()

	assertion := h.AssertEvent(t, "registration.RegistrationStarted")

	var event registration.RegistrationStarted
	err := json.Unmarshal(assertion.payload, &event)
	require.NoError(t, err)

	return &RegistrationStartedAssertion{
		EventAssertion: assertion,
		event:          event,
	}
}

type EventAssertion struct {
	t         *testing.T
	eventType string
	payload   json.RawMessage
	metadata  json.RawMessage
	offset    int64
}

func (a *EventAssertion) Parse(event any) *EventAssertion {
	a.t.Helper()
	err := json.Unmarshal(a.payload, event)
	require.NoError(a.t, err, "failed to parse event payload")
	return a
}

func (a *EventAssertion) HasField(field string, expected any) *EventAssertion {
	a.t.Helper()

	var data map[string]any
	err := json.Unmarshal(a.payload, &data)
	require.NoError(a.t, err)

	actual, exists := data[field]
	require.True(a.t, exists, "field %s not found in event", field)
	assert.Equal(a.t, expected, actual, "unexpected value for field %s", field)

	return a
}

func (a *EventAssertion) GetPayload() json.RawMessage {
	return a.payload
}

type RegistrationStartedAssertion struct {
	*EventAssertion
	event registration.RegistrationStarted
}

func (a *RegistrationStartedAssertion) HasEmail(expected string) *RegistrationStartedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.event.Email, "unexpected email in event")
	return a
}

func (a *RegistrationStartedAssertion) HasVerificationCode() *RegistrationStartedAssertion {
	a.t.Helper()
	assert.NotEmpty(a.t, a.event.VerificationCode, "verification code should not be empty")
	return a
}

func (a *RegistrationStartedAssertion) HasRegistrationID(expected registration.ID) *RegistrationStartedAssertion {
	a.t.Helper()
	assert.Equal(a.t, expected, a.event.RegistrationID, "unexpected registration ID")
	return a
}

func (a *RegistrationStartedAssertion) GetVerificationCode() string {
	return a.event.VerificationCode
}

func (h *Helper) GetEventStream(t *testing.T, streamName string) []EventRecord {
	t.Helper()

	query := fmt.Sprintf(`
        SELECT "offset", payload, metadata
        FROM watermill_%s
        ORDER BY "offset"
    `, streamName)

	rows, err := h.pool.Query(context.Background(), query)
	require.NoError(t, err)
	defer rows.Close()

	var events []EventRecord
	for rows.Next() {
		var e EventRecord
		err := rows.Scan(&e.Offset, &e.Payload, &e.Metadata)
		require.NoError(t, err)
		events = append(events, e)
	}

	return events
}

type EventRecord struct {
	Offset   int64
	Payload  json.RawMessage
	Metadata json.RawMessage
}

func (h *Helper) ClearAllEvents(t *testing.T) {
	t.Helper()

	tables := []string{
		"watermill_events_registration",
		"watermill_offsets_events_registration",
	}

	for _, table := range tables {
		_, err := h.pool.Exec(context.Background(), fmt.Sprintf("TRUNCATE TABLE %s", table))
		require.NoError(t, err)
	}
}
