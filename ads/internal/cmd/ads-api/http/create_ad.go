package http

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

type createAdBody struct {
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
	PhotoURL   string `json:"photo_url"`
}

type createAdResponse struct {
	ID int64 `json:"id"`
}

// handlerCreateAd serves POST /ads.
func handlerCreateAd(createAd usecase.CreateAdFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var body createAdBody
		if err := decodeBody(r, &body); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		adID, err := createAd(r.Context(), usecase.CreateAdRequest{
			Title:      body.Title,
			PriceCents: body.PriceCents,
			PhotoURL:   body.PhotoURL,
			OwnerID:    middleware.UserIDFromRequest(r),
		})
		if err != nil {
			if quotaErr, ok := errors.AsType[usecase.QuotaExceededError](err); ok {
				rw.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds(quotaErr.RetryAt)))
				rw.WriteHeader(http.StatusTooManyRequests)
				return
			}

			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(createAdResponse{ID: int64(adID)})
	}
}

// retryAfterSeconds rounds the delay up to whole seconds, with a floor of one so a client never retries immediately.
func retryAfterSeconds(at time.Time) int {
	return max(1, int(math.Ceil(time.Until(at).Seconds())))
}
