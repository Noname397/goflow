package job

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewJobGeneratesUUIDv4(t *testing.T) {
	created, err := NewJob("compute", nil)
	if err != nil {
		t.Fatalf("NewJob() error = %v", err)
	}

	id, err := uuid.Parse(created.ID)
	if err != nil {
		t.Fatalf("NewJob() ID %q is not a valid UUID: %v", created.ID, err)
	}
	if id.Version() != uuid.Version(4) {
		t.Errorf("NewJob() ID %q is not UUID version 4", created.ID)
	}
	if id.Variant() != uuid.RFC4122 {
		t.Errorf("NewJob() ID %q does not use the RFC 4122 variant", created.ID)
	}
}
