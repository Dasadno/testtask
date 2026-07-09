package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/Dasadno/testtask/internal/models"
	transport "github.com/Dasadno/testtask/internal/transport/http"
)

// serviceMock — ручной мок SubscriptionService.
type serviceMock struct {
	createFn func(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	getFn    func(ctx context.Context, id uuid.UUID) (models.Subscription, error)
	updateFn func(ctx context.Context, sub models.Subscription) (models.Subscription, error)
	deleteFn func(ctx context.Context, id uuid.UUID) error
	listFn   func(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error)
}

func (m *serviceMock) Create(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	return m.createFn(ctx, sub)
}

func (m *serviceMock) Get(ctx context.Context, id uuid.UUID) (models.Subscription, error) {
	return m.getFn(ctx, id)
}

func (m *serviceMock) Update(ctx context.Context, sub models.Subscription) (models.Subscription, error) {
	return m.updateFn(ctx, sub)
}

func (m *serviceMock) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func (m *serviceMock) List(ctx context.Context, filter models.SubscriptionFilter) ([]models.Subscription, error) {
	return m.listFn(ctx, filter)
}

func newTestRouter(svc *serviceMock) http.Handler {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return transport.NewRouter(log, "test", transport.NewSubscriptionHandler(svc, log))
}

func doRequest(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req := httptest.NewRequest(method, target, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCreateSubscription_OK(t *testing.T) {
	var passed models.Subscription
	svc := &serviceMock{
		createFn: func(_ context.Context, sub models.Subscription) (models.Subscription, error) {
			passed = sub
			sub.ID = uuid.New()
			return sub, nil
		},
	}

	body := `{"service_name":"Yandex Plus","price":400,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}`
	rec := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/subscriptions", body)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
	}
	if passed.ServiceName != "Yandex Plus" || passed.Price != 400 {
		t.Errorf("service received %+v", passed)
	}
	if passed.StartDate.String() != "07-2025" {
		t.Errorf("StartDate = %s, want 07-2025", passed.StartDate)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response json: %v", err)
	}
	if resp["start_date"] != "07-2025" {
		t.Errorf("response start_date = %v, want 07-2025", resp["start_date"])
	}
	if _, hasEnd := resp["end_date"]; hasEnd {
		t.Error("end_date must be omitted when not set")
	}
}

func TestCreateSubscription_BadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `{`},
		{"missing price", `{"service_name":"X","user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}`},
		{"invalid uuid", `{"service_name":"X","price":1,"user_id":"not-a-uuid","start_date":"07-2025"}`},
		{"invalid date format", `{"service_name":"X","price":1,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"2025-07"}`},
		{"numeric date", `{"service_name":"X","price":1,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":42}`},
	}

	svc := &serviceMock{
		createFn: func(context.Context, models.Subscription) (models.Subscription, error) {
			t.Fatal("service must not be called on bad request")
			return models.Subscription{}, nil
		},
	}
	router := newTestRouter(svc)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, router, http.MethodPost, "/api/v1/subscriptions", tt.body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
		})
	}
}

func TestCreateSubscription_ValidationErrorMapsTo400(t *testing.T) {
	svc := &serviceMock{
		createFn: func(context.Context, models.Subscription) (models.Subscription, error) {
			return models.Subscription{}, fmt.Errorf("%w: end_date must not be before start_date", models.ErrInvalidSubscription)
		},
	}

	body := `{"service_name":"X","price":1,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025","end_date":"01-2025"}`
	rec := doRequest(t, newTestRouter(svc), http.MethodPost, "/api/v1/subscriptions", body)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "end_date") {
		t.Errorf("body must explain the error, got: %s", rec.Body)
	}
}

func TestGetSubscription_NotFound(t *testing.T) {
	svc := &serviceMock{
		getFn: func(context.Context, uuid.UUID) (models.Subscription, error) {
			return models.Subscription{}, fmt.Errorf("get: %w", models.ErrSubscriptionNotFound)
		},
	}

	rec := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/subscriptions/"+uuid.NewString(), "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404; body: %s", rec.Code, rec.Body)
	}
}

func TestGetSubscription_InvalidID(t *testing.T) {
	svc := &serviceMock{
		getFn: func(context.Context, uuid.UUID) (models.Subscription, error) {
			t.Fatal("service must not be called for invalid id")
			return models.Subscription{}, nil
		},
	}

	rec := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/subscriptions/abc", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestGetSubscription_InternalErrorHidesDetails(t *testing.T) {
	svc := &serviceMock{
		getFn: func(context.Context, uuid.UUID) (models.Subscription, error) {
			return models.Subscription{}, fmt.Errorf("pg: connection refused on 10.0.0.1")
		},
	}

	rec := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/subscriptions/"+uuid.NewString(), "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "10.0.0.1") {
		t.Errorf("internal details leaked to client: %s", rec.Body)
	}
}

func TestDeleteSubscription_NoContent(t *testing.T) {
	svc := &serviceMock{
		deleteFn: func(context.Context, uuid.UUID) error { return nil },
	}

	rec := doRequest(t, newTestRouter(svc), http.MethodDelete, "/api/v1/subscriptions/"+uuid.NewString(), "")
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestUpdateSubscription_PassesPathID(t *testing.T) {
	id := uuid.New()
	svc := &serviceMock{
		updateFn: func(_ context.Context, sub models.Subscription) (models.Subscription, error) {
			if sub.ID != id {
				t.Errorf("service received id %s, want %s from path", sub.ID, id)
			}
			return sub, nil
		},
	}

	body := `{"service_name":"X","price":1,"user_id":"60601fee-2bf1-4721-ae6f-7636e79a0cba","start_date":"07-2025"}`
	rec := doRequest(t, newTestRouter(svc), http.MethodPut, "/api/v1/subscriptions/"+id.String(), body)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; body: %s", rec.Code, rec.Body)
	}
}

func TestListSubscriptions(t *testing.T) {
	userID := uuid.New()

	t.Run("filters passed to service", func(t *testing.T) {
		var got models.SubscriptionFilter
		svc := &serviceMock{
			listFn: func(_ context.Context, f models.SubscriptionFilter) ([]models.Subscription, error) {
				got = f
				return nil, nil
			},
		}

		target := "/api/v1/subscriptions?user_id=" + userID.String() + "&service_name=Netflix&limit=10&offset=5"
		rec := doRequest(t, newTestRouter(svc), http.MethodGet, target, "")

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}
		if got.UserID == nil || *got.UserID != userID {
			t.Errorf("filter.UserID = %v, want %s", got.UserID, userID)
		}
		if got.ServiceName == nil || *got.ServiceName != "Netflix" {
			t.Errorf("filter.ServiceName = %v, want Netflix", got.ServiceName)
		}
		if got.Limit != 10 || got.Offset != 5 {
			t.Errorf("pagination = %d/%d, want 10/5", got.Limit, got.Offset)
		}
	})

	t.Run("limit=0 treated as unset and passed through", func(t *testing.T) {
		// omitempty: нулевой limit не считается ошибкой — сервис подставит дефолт.
		var got models.SubscriptionFilter
		svc := &serviceMock{
			listFn: func(_ context.Context, f models.SubscriptionFilter) ([]models.Subscription, error) {
				got = f
				return nil, nil
			},
		}

		rec := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/subscriptions?limit=0", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got.Limit != 0 {
			t.Errorf("filter.Limit = %d, want 0 (unset)", got.Limit)
		}
	})

	t.Run("empty result is json array", func(t *testing.T) {
		svc := &serviceMock{
			listFn: func(context.Context, models.SubscriptionFilter) ([]models.Subscription, error) {
				return nil, nil
			},
		}

		rec := doRequest(t, newTestRouter(svc), http.MethodGet, "/api/v1/subscriptions", "")
		if !strings.Contains(rec.Body.String(), `"items":[]`) {
			t.Errorf("items must be [] not null: %s", rec.Body)
		}
	})

	t.Run("invalid query params", func(t *testing.T) {
		svc := &serviceMock{
			listFn: func(context.Context, models.SubscriptionFilter) ([]models.Subscription, error) {
				t.Fatal("service must not be called")
				return nil, nil
			},
		}

		for _, target := range []string{
			"/api/v1/subscriptions?user_id=oops",
			"/api/v1/subscriptions?limit=101",
			"/api/v1/subscriptions?offset=-1",
		} {
			rec := doRequest(t, newTestRouter(svc), http.MethodGet, target, "")
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400", target, rec.Code)
			}
		}
	})
}
