package http

import (
	"encoding/json"
	"net/http"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
)

type importAdsBody struct {
	Ads []createAdBody `json:"ads"`
}

type importAdsResponse struct {
	IDs []int64 `json:"ids"`
}

func handlerImportAds(importAds usecase.ImportAdsFunc) http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		var body importAdsBody
		if err := decodeBody(r, &body); err != nil {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		ownerID := middleware.UserIDFromRequest(r)
		reqs := make([]usecase.CreateAdRequest, 0, len(body.Ads))
		for _, item := range body.Ads {
			reqs = append(reqs, usecase.CreateAdRequest{
				Title:      item.Title,
				PriceCents: item.PriceCents,
				PhotoURL:   item.PhotoURL,
				OwnerID:    ownerID,
			})
		}

		ids := make([]int64, 0, len(reqs))
		if len(reqs) > 0 {
			created, err := importAds(r.Context(), reqs)
			if err != nil {
				rw.WriteHeader(http.StatusInternalServerError)
				return
			}

			for _, id := range created {
				ids = append(ids, int64(id))
			}
		}

		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(rw).Encode(importAdsResponse{IDs: ids})
	}
}
