package usecase

import (
	"context"
	"fmt"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

type (
	// DeleteAdFunc deletes an ad on behalf of its owner. It fails with adsdao.ErrNotFound when the ad does not exist or belongs to another user.
	DeleteAdFunc func(ctx context.Context, id domain.AdID, ownerID domain.UserID) error

	deleteAd struct {
		adDAO adsdao.DAO
	}
)

// NewDeleteAdFunc returns a DeleteAdFunc backed by the given DAO.
func NewDeleteAdFunc(adDAO adsdao.DAO) DeleteAdFunc {
	return deleteAd{adDAO: adDAO}.deleteAd
}

func (uc deleteAd) deleteAd(ctx context.Context, id domain.AdID, ownerID domain.UserID) error {
	if err := uc.adDAO.DeleteAd(ctx, id, ownerID); err != nil {
		return fmt.Errorf("unable to delete ad %s: %w", id, err)
	}

	return nil
}
