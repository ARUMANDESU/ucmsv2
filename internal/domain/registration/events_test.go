package registration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ARUMANDESU/ucms/internal/domain/event"
)

func TestRegistrationID_JSONMarshaling(t *testing.T) {
	// Create a registration ID
	originalID := NewID()

	// Marshal to JSON
	data, err := json.Marshal(originalID)
	require.NoError(t, err)

	// Should be marshaled as a string, not a byte array
	var str string
	err = json.Unmarshal(data, &str)
	require.NoError(t, err)
	assert.Equal(t, originalID.String(), str)

	// Unmarshal back
	var unmarshaledID ID
	err = json.Unmarshal(data, &unmarshaledID)
	require.NoError(t, err)

	// Should be equal
	assert.Equal(t, originalID, unmarshaledID)
}

func TestRegistrationStartedEvent_JSONMarshaling(t *testing.T) {
	// Create an event
	originalEvent := &RegistrationStarted{
		Header:           event.NewEventHeader(),
		Otel:             event.Otel{Carrier: map[string]string{"trace": "123"}},
		RegistrationID:   NewID(),
		Email:            "test@example.com",
		VerificationCode: "ABC123",
	}

	// Marshal to JSON
	data, err := json.Marshal(originalEvent)
	require.NoError(t, err)

	// Unmarshal back
	var unmarshaledEvent RegistrationStarted
	err = json.Unmarshal(data, &unmarshaledEvent)
	require.NoError(t, err)

	// Should have the same values
	assert.Equal(t, originalEvent.RegistrationID, unmarshaledEvent.RegistrationID)
	assert.Equal(t, originalEvent.Email, unmarshaledEvent.Email)
	assert.Equal(t, originalEvent.VerificationCode, unmarshaledEvent.VerificationCode)
	assert.Equal(t, originalEvent.Carrier, unmarshaledEvent.Carrier)
}
