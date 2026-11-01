package job

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// JobStatus represents the lifecycle state of a job.
type JobStatus string

const (
	StatusPending JobStatus = "pending"
	StatusRunning JobStatus = "running"
	StatusSuccess JobStatus = "succeeded"
	StatusFailed  JobStatus = "failed"
)

// Job represents a queued workload.
type Job struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Payload      map[string]any `json:"payload"`
	Status       JobStatus      `json:"status"`
	Result       any            `json:"result,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	StartedAt    *time.Time     `json:"started_at,omitempty"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
}

// NewJob validates and creates a job instance.
func NewJob(jobType string, payload map[string]any) (Job, error) {
	if jobType == "" {
		return Job{}, fmt.Errorf("job type is required")
	}

	switch jobType {
	case "sleep", "compute":
	default:
		return Job{}, fmt.Errorf("unsupported job type %q", jobType)
	}

	if payload == nil {
		payload = map[string]any{}
	}

	return Job{
		ID:        newID(),
		Type:      jobType,
		Payload:   payload,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// MarshalJSON is used to normalize empty payloads and time fields.
func (j Job) MarshalJSON() ([]byte, error) {
	type Alias Job
	data := struct {
		Alias
		Payload map[string]any `json:"payload"`
	}{
		Alias:   Alias(j),
		Payload: j.Payload,
	}
	if data.Payload == nil {
		data.Payload = map[string]any{}
	}
	return json.Marshal(data)
}

func newID() string {
	return uuid.New().String()
}
