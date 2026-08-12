package comm

import (
	"errors"
	"testing"
)

func TestFindCountInvalidDataType(t *testing.T) {
	// Test that findCount returns ErrInvalidDataType for invalid inputs
	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "nil data",
			data: nil,
		},
		{
			name: "non-pointer",
			data: 42,
		},
		{
			name: "pointer to non-slice",
			data: &struct{}{},
		},
		{
			name: "pointer to struct",
			data: &struct{ Name string }{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need a database session to test findCount
			// Since we're only testing the validation logic, we can use a nil session
			// The validation happens before any database operations
			_, err := findCount(nil, tt.data)

			if err == nil {
				t.Fatal("findCount() expected error for invalid data, got nil")
			}

			if !errors.Is(err, ErrInvalidDataType) {
				t.Errorf("findCount() error = %v, want errors.Is(err, ErrInvalidDataType)", err)
			}
		})
	}
}
