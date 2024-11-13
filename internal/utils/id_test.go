package utils

import (
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int // expected length
	}{
		{
			name:   "generates 8 character ID",
			length: 8,
			want:   8,
		},
		{
			name:   "generates 16 character ID",
			length: 16,
			want:   16,
		},
		{
			name:   "generates 32 character ID",
			length: 32,
			want:   32,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MustGenerateID(tt.length)
			if len(got) != tt.want {
				t.Errorf("GenerateID() length = %v, want %v", len(got), tt.want)
			}

			// Test uniqueness by generating another ID
			got2 := MustGenerateID(tt.length)
			if got == got2 {
				t.Error("GenerateID() generated duplicate IDs")
			}
		})
	}
}
