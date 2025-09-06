package mocks

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/ucmsv2/ucms-backend/internal/domain/valueobject/mails"
)

type MockMailSender struct {
	mu        sync.Mutex
	sentMails []mails.Payload
}

func NewMockMailSender() *MockMailSender {
	return &MockMailSender{
		sentMails: make([]mails.Payload, 0),
	}
}

func (m *MockMailSender) SendMail(ctx context.Context, payload mails.Payload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentMails = append(m.sentMails, mails.Payload{
		To:      payload.To,
		Subject: payload.Subject,
		Body:    payload.Body,
	})
	slog.Debug("MockMailSender: SendMail called", "to", payload.To, "subject", payload.Subject, "body", payload.Body)
	return nil
}

func (m *MockMailSender) GetSentMails() []mails.Payload {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]mails.Payload{}, m.sentMails...)
}

func (m *MockMailSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentMails = make([]mails.Payload, 0)
}

func (m *MockMailSender) AssertMailSent(t *testing.T, email, subject string) {
	sentMails := m.GetSentMails()
	for _, mail := range sentMails {
		if mail.To == email && strings.Contains(mail.Subject, subject) {
			return
		}
	}
	t.Errorf("Expected mail to %s with subject containing %s not found", email, subject)
}

// EventuallyRequireMailSent checks periodically for up to 5 seconds if an email with the specified subject has been sent to the given address.
func (m *MockMailSender) EventuallyRequireMailSent(t *testing.T, email, subject string) *mails.Payload {
	t.Helper()
	var foundMail mails.Payload
	require.Eventually(t, func() bool {
		sentMails := m.GetSentMails()
		for _, mail := range sentMails {
			if mail.To == email && strings.Contains(mail.Subject, subject) {
				foundMail = mail
				return true
			}
		}
		return false
	}, 5*time.Second, 100*time.Millisecond, "Expected mail to %s with subject containing %s not found within timeout", email, subject)
	return &foundMail
}
