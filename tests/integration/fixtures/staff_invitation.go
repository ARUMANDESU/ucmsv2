package fixtures

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	StaffInvitationAcceptPageURL = "http://localhost:3000/invitations/staff/accept"
	InvitationTokenKey           = "invitation_test_key"
	InvitationTokenExp           = 15 * time.Minute

	StaffInvitationValidCode   = "F0WNPKO98NOGYVC5BPOZ"
	StaffInvitationInvalidCode = "INVALIDCODE123456789"
)

var InvitationTokenAlg = jwt.SigningMethodHS256
