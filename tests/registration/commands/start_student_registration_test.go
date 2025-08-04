package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/ARUMANDESU/ucms/tests/integration"
)

type CommandSuite struct {
	integration.TestSuite
}

func (s *CommandSuite) TestStartStudentRegistration() {
	email := "student@test.com"

	s.Run("Start Student Registration", func() {
		reqBody := map[string]string{"email": email}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/v1/registration/start/student", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		s.App().HTTPHandler.ServeHTTP(w, req)

		require.Equal(s.T(), http.StatusAccepted, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(s.T(), err)
		assert.True(s.T(), response["succeeded"].(bool))

		s.AssertRegistrationExists(email)

		s.AssertRegistrationStartedEvent(email)
	})
}

func TestCommandSuite(t *testing.T) {
	suite.Run(t, new(CommandSuite))
}
