package usecase

import (
	"context"
	"fmt"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type (
	// ImportAdsFunc creates every ad of the batch, or none of them: the batch runs in one transaction, so a failure on any item leaves nothing behind and a retry starts from a clean state. The caller sets OwnerID on every item from the authenticated user.
	ImportAdsFunc func(ctx context.Context, reqs []CreateAdRequest) ([]domain.AdID, error)

	importAds struct {
		adDAO adsdao.DAO
	}
)

// NewImportAdsFunc returns an ImportAdsFunc backed by the given DAO.
func NewImportAdsFunc(adDAO adsdao.DAO) ImportAdsFunc {
	return importAds{adDAO: adDAO}.importAds
}

func (uc importAds) importAds(ctx context.Context, reqs []CreateAdRequest) ([]domain.AdID, error) {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to import ads: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ids := make([]domain.AdID, 0, len(reqs))
	for i, req := range reqs {
		id, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
			Title:      req.Title,
			PriceCents: req.PriceCents,
			PhotoURL:   req.PhotoURL,
			OwnerID:    req.OwnerID,
		})
		if err != nil {
			return nil, fmt.Errorf("unable to import ad %d: %w", i, err)
		}

		ids = append(ids, id)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("unable to import ads: %w", err)
	}

	return ids, nil
}
