package role

import "testing"

func TestIsGlobalValid(t *testing.T) {
	tests := []struct {
		role  string
		valid bool
	}{
		{"guest", true},
		{"student", true},
		{"aitusa", true},
		{"staff", true},
		{"invalid", false},
		{"", false},
		{"GuestRole", false},
		{"StudentRole", false},
		{"AITUSARole", false},
		{"StaffRole", false},
		{"Guest", false},
		{"Student", false},
		{"AITUSA", false},
		{"Staff", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if IsGlobalValid(tt.role) != tt.valid {
				t.Errorf("IsGlobalValid(%q) = %v; want %v", tt.role, !tt.valid, tt.valid)
			}
		})
	}
}
