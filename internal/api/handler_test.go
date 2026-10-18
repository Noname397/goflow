package api

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "goflow/internal/job"
    "goflow/internal/store"
)

func TestCreateAndListJobs(t *testing.T) {
    svc := job.NewService(store.NewMemoryStore())
    h := NewHandler(svc)

    req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"type":"sleep","payload":{"duration_ms":5}}`))
    res := httptest.NewRecorder()
    h.ServeHTTP(res, req)
    if res.Code != http.StatusCreated {
        t.Fatalf("unexpected create status: %d body=%s", res.Code, res.Body.String())
    }

    var created job.Job
    if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
        t.Fatal(err)
    }
    if created.ID == "" {
        t.Fatal("expected job ID")
    }

    listReq := httptest.NewRequest(http.MethodGet, "/jobs?status=pending", nil)
    listRes := httptest.NewRecorder()
    h.ServeHTTP(listRes, listReq)
    if listRes.Code != http.StatusOK {
        t.Fatalf("unexpected list status: %d body=%s", listRes.Code, listRes.Body.String())
    }

    getReq := httptest.NewRequest(http.MethodGet, "/jobs/"+created.ID, nil)
    getRes := httptest.NewRecorder()
    h.ServeHTTP(getRes, getReq)
    if getRes.Code != http.StatusOK {
        t.Fatalf("unexpected get status: %d body=%s", getRes.Code, getRes.Body.String())
    }
}
