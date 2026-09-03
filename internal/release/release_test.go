package release

import "testing"

func TestPolicyValid(t *testing.T) {
	tests := []struct {
		policy Policy
		want   bool
	}{
		{PolicyOptional, true},
		{PolicyRequired, true},
		{Policy(""), false},
		{Policy("mandatory"), false},
	}

	for _, tt := range tests {
		if got := tt.policy.Valid(); got != tt.want {
			t.Errorf("Policy(%q).Valid() = %v, want %v", tt.policy, got, tt.want)
		}
	}
}
