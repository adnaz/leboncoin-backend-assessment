package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

func newImportAdsServer(fn usecase.ImportAdsFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.Handle("POST /ads/import", middleware.RequireAuth(handlerImportAds(fn)))
	return httptest.NewServer(mux)
}

func TestHandlerImportAds(t *testing.T) {
	t.Run("returns 201 with all generated IDs on success", func(t *testing.T) {
		srv := newImportAdsServer(func(_ context.Context, reqs []usecase.CreateAdRequest) ([]domain.AdID, error) {
			require.Len(t, reqs, 3)
			return []domain.AdID{10, 11, 12}, nil
		})
		defer srv.Close()

		body := importAdsBody{Ads: []createAdBody{
			{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg"},
			{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg"},
			{Title: "car", PriceCents: 200000, PhotoURL: "https://example.com/car.jpg"},
		}}
		resp := postJSON(t, srv.URL+"/ads/import", body, "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var got importAdsResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Equal(t, []int64{10, 11, 12}, got.IDs)
	})

	t.Run("passes authenticated user ID as owner for every ad", func(t *testing.T) {
		var received []usecase.CreateAdRequest
		srv := newImportAdsServer(func(_ context.Context, reqs []usecase.CreateAdRequest) ([]domain.AdID, error) {
			received = reqs
			return []domain.AdID{1, 2}, nil
		})
		defer srv.Close()

		body := importAdsBody{Ads: []createAdBody{
			{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg"},
			{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg"},
		}}
		resp := postJSON(t, srv.URL+"/ads/import", body, "user-42")
		defer resp.Body.Close()

		require.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Len(t, received, 2)
		assert.Equal(t, domain.UserID("user-42"), received[0].OwnerID)
		assert.Equal(t, domain.UserID("user-42"), received[1].OwnerID)
	})

	t.Run("returns 201 with empty IDs when body has no ads", func(t *testing.T) {
		srv := newImportAdsServer(func(_ context.Context, _ []usecase.CreateAdRequest) ([]domain.AdID, error) {
			t.Fatal("usecase should not be called")
			return nil, nil
		})
		defer srv.Close()

		resp := postJSON(t, srv.URL+"/ads/import", importAdsBody{Ads: []createAdBody{}}, "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)

		var got importAdsResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&got))
		assert.Empty(t, got.IDs)
	})

	t.Run("returns 401 when Authorization header is missing", func(t *testing.T) {
		srv := newImportAdsServer(func(_ context.Context, _ []usecase.CreateAdRequest) ([]domain.AdID, error) {
			t.Fatal("usecase should not be called")
			return nil, nil
		})
		defer srv.Close()

		resp := postJSON(t, srv.URL+"/ads/import", importAdsBody{}, "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("returns 400 on invalid body", func(t *testing.T) {
		srv := newImportAdsServer(func(_ context.Context, _ []usecase.CreateAdRequest) ([]domain.AdID, error) {
			t.Fatal("usecase should not be called")
			return nil, nil
		})
		defer srv.Close()

		req, err := http.NewRequest(http.MethodPost, srv.URL+"/ads/import", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "user-1")

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("returns 500 and no IDs when the import fails", func(t *testing.T) {
		srv := newImportAdsServer(func(_ context.Context, _ []usecase.CreateAdRequest) ([]domain.AdID, error) {
			return nil, errors.New("db down")
		})
		defer srv.Close()

		body := importAdsBody{Ads: []createAdBody{
			{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg"},
			{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg"},
			{Title: "car", PriceCents: 200000, PhotoURL: "https://example.com/car.jpg"},
		}}
		resp := postJSON(t, srv.URL+"/ads/import", body, "user-1")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}
