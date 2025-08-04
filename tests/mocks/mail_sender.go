package mocks

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/ARUMANDESU/ucms/internal/domain/valueobject/mail"
)

type MockMailSender struct {
	mu        sync.Mutex
	sentMails []mail.Payload
}

func NewMockMailSender() *MockMailSender {
	return &MockMailSender{
		sentMails: make([]mail.Payload, 0),
	}
}

func (m *MockMailSender) SendMail(ctx context.Context, payload mail.Payload) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentMails = append(m.sentMails, mail.Payload{
		To:      payload.To,
		Subject: payload.Subject,
		Body:    payload.Body,
	})
	return nil
}

func (m *MockMailSender) GetSentMails() []mail.Payload {
	m.mu.Lock()
	defer m.mu.Unlock()

	return append([]mail.Payload{}, m.sentMails...)
}

func (m *MockMailSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sentMails = make([]mail.Payload, 0)
}

func (m *MockMailSender) AssertMailSent(t *testing.T, email, subject string) {
	mails := m.GetSentMails()
	for _, mail := range mails {
		if mail.To == email && strings.Contains(mail.Subject, subject) {
			return
		}
	}
	t.Errorf("Expected mail to %s with subject containing %s not found", email, subject)
}
