package models_test

import (
	"testing"

	"github.com/GagarinRu/avatars/internal/models"
)

func TestValidStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		status models.ProcessingStatus
		want   bool
	}{
		{models.StatusPending, true},
		{models.StatusProcessing, true},
		{models.StatusReady, true},
		{models.StatusFailed, true},
		{models.ProcessingStatus("unknown"), false},
	}
	for _, tc := range cases {
		if got := models.ValidStatus(tc.status); got != tc.want {
			t.Fatalf("ValidStatus(%q) = %v, want %v", tc.status, got, tc.want)
		}
	}
}
