package api

import (
    "context"
    "encoding/json"
    "errors"
    "net/http"
    "strings"

    "goflow/internal/job"
)

// JobRequest defines the payload accepted by the HTTP API.
type JobRequest struct {
    Type    string         `json:"type"`
    Payload map[string]any `json:"payload"`
}

// Handler routes HTTP requests.
type Handler struct {
    service   *job.Service
    processor func(context.Context, job.Job) (job.Job, error)
}

func NewHandler(service *job.Service, processor ...func(context.Context, job.Job) (job.Job, error)) *Handler {
    h := &Handler{service: service}
    if len(processor) > 0 {
        h.processor = processor[0]
    }
    return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    switch {
    case r.Method == http.MethodGet && r.URL.Path == "/health":
        h.handleHealth(w, r)
    case r.Method == http.MethodPost && r.URL.Path == "/jobs":
        h.handleCreateJob(w, r)
    case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/jobs/"):
        h.handleGetJob(w, r)
    case r.Method == http.MethodGet && r.URL.Path == "/jobs":
        h.handleListJobs(w, r)
    default:
        http.NotFound(w, r)
    }
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleCreateJob(w http.ResponseWriter, r *http.Request) {
    var req JobRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid JSON body", http.StatusBadRequest)
        return
    }
    if req.Type == "" {
        http.Error(w, "job type is required", http.StatusBadRequest)
        return
    }

    created, err := h.service.Submit(r.Context(), req.Type, req.Payload)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnprocessableEntity)
        return
    }
    if h.processor != nil {
        processed, err := h.processor(r.Context(), created)
        if err != nil {
            processed = created
            processed.Status = job.StatusFailed
            processed.ErrorMessage = err.Error()
            if _, updateErr := h.service.MarkFailed(r.Context(), processed, err.Error()); updateErr != nil {
                http.Error(w, updateErr.Error(), http.StatusInternalServerError)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(http.StatusCreated)
            _ = json.NewEncoder(w).Encode(processed)
            return
        }
        created = processed
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    _ = json.NewEncoder(w).Encode(created)
}

func (h *Handler) handleGetJob(w http.ResponseWriter, r *http.Request) {
    id := strings.TrimPrefix(r.URL.Path, "/jobs/")
    if id == "" || id == "/jobs/" {
        http.Error(w, "job id is required", http.StatusBadRequest)
        return
    }
    jobItem, err := h.service.GetByID(r.Context(), id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(jobItem)
}

func (h *Handler) handleListJobs(w http.ResponseWriter, r *http.Request) {
    status := r.URL.Query().Get("status")
    var filter []job.JobStatus
    if status != "" {
        filter = append(filter, job.JobStatus(status))
    }
    jobs, err := h.service.List(context.Background(), filter...)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(jobs)
}

func (h *Handler) handleHealthWithContext(ctx context.Context) error {
    return errors.New("not implemented")
}
