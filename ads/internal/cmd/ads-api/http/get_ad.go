package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

type getAdResponse struct {
	ID         int64  `json:"id"`
	Title      string `json:"title"`
	PriceCents int64  `json:"price_cents"`
	PhotoURL   string `json:"photo_url"`
	UserID     string `json:"user_id"`
	// Username is null when the identity service could not provide it.
	Username  *string   `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

func adToGetAdResponse(a usecase.AdWithOwner) getAdResponse {
	return getAdResponse{
		ID:         int64(a.Ad.ID),
		Title:      a.Ad.Title,
		PriceCents: a.Ad.Price,
		PhotoURL:   a.Ad.PhotoURL,
		UserID:     a.Ad.UserID.String(),
		Username:   a.Username,
		CreatedAt:  a.Ad.CreatedAt.UTC(),
	}
}

func handlerGetAd(getAd usecase.GetAdByIDFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		adID, err := adIDFromURL(r)
		if err != nil {
			rw.WriteHeader(http.StatusBadRequest)

			return
		}

		ad, err := getAd(r.Context(), adID)
		if err != nil {
			if errors.Is(err, adsdao.ErrNotFound) {
				rw.WriteHeader(http.StatusNotFound)
			} else {
				rw.WriteHeader(http.StatusInternalServerError)
			}

			return
		}

		rw.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(rw).Encode(adToGetAdResponse(ad))
	}
}
