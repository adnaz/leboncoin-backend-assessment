package usecase

import (
	"context"
	"fmt"
	"time"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

const (
	// adQuota is the number of ads a user can publish through POST /ads within adQuotaWindow.
	adQuota       = 5
	adQuotaWindow = 24 * time.Hour
)

type (
	// CreateAdFunc creates an ad owned by the calling user, or returns a QuotaExceededError when the user already published adQuota ads in the last adQuotaWindow.
	CreateAdFunc func(ctx context.Context, req CreateAdRequest) (domain.AdID, error)

	// CreateAdRequest describes the ad to create. OwnerID is the authenticated user: a caller can never create an ad on behalf of someone else.
	CreateAdRequest struct {
		Title      string
		PriceCents int64
		PhotoURL   string
		OwnerID    domain.UserID
	}

	// QuotaExceededError is returned when the user reached the quota. RetryAt is the earliest time a new ad can be published.
	QuotaExceededError struct {
		RetryAt time.Time
	}

	createAd struct {
		adDAO adsdao.DAO
	}
)

// Error implements the error interface.
func (e QuotaExceededError) Error() string {
	return "ad quota exceeded, retry at " + e.RetryAt.Format(time.RFC3339)
}

// NewCreateAdFunc returns a CreateAdFunc backed by the given DAO.
func NewCreateAdFunc(adDAO adsdao.DAO) CreateAdFunc {
	return createAd{adDAO: adDAO}.createAd
}

// createAd counts and inserts in one transaction under a per-owner lock, so two concurrent requests from the same user cannot both pass the count. Lock, count and insert stay three statements on purpose: see dao.Tx.LockOwner.
func (uc createAd) createAd(ctx context.Context, req CreateAdRequest) (domain.AdID, error) {
	tx, err := uc.adDAO.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.LockOwner(ctx, req.OwnerID); err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	recent, err := tx.RecentAdCreations(ctx, req.OwnerID, adQuotaWindow, adQuota)
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	if len(recent) >= adQuota {
		return 0, QuotaExceededError{RetryAt: recent[len(recent)-1].Add(adQuotaWindow)}
	}

	adID, err := tx.CreateAd(ctx, adsdao.CreateAdRequest{
		Title:      req.Title,
		PriceCents: req.PriceCents,
		PhotoURL:   req.PhotoURL,
		OwnerID:    req.OwnerID,
	})
	if err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("unable to create ad: %w", err)
	}

	return adID, nil
}
