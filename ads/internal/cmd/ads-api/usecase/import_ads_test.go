package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	mockdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestImportAds(t *testing.T) {
	ctx := context.Background()
	reqs := []usecase.CreateAdRequest{
		{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg", OwnerID: "user-1"},
		{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg", OwnerID: "user-1"},
	}
	daoReqs := []adsdao.CreateAdRequest{
		{Title: "bike", PriceCents: 5000, PhotoURL: "https://example.com/bike.jpg", OwnerID: "user-1"},
		{Title: "scooter", PriceCents: 8000, PhotoURL: "https://example.com/scooter.jpg", OwnerID: "user-1"},
	}

	t.Run("creates every ad in one transaction", func(t *testing.T) {
		m := mockdao.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, daoReqs[0]).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, daoReqs[1]).Return(domain.AdID(11), nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		require.NoError(t, err)
		assert.Equal(t, []domain.AdID{10, 11}, got)
	})

	t.Run("rolls back and creates nothing when one insert fails", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, daoReqs[0]).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, daoReqs[1]).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		require.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("returns the error when the transaction cannot be opened", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("db down")
		m.On("Begin", ctx).Return(dbErr)

		_, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		assert.ErrorIs(t, err, dbErr)
	})

	t.Run("returns the error when commit fails", func(t *testing.T) {
		m := mockdao.New(t)
		dbErr := errors.New("commit failed")
		m.On("Begin", ctx).Return(nil)
		m.On("CreateAd", ctx, daoReqs[0]).Return(domain.AdID(10), nil)
		m.On("CreateAd", ctx, daoReqs[1]).Return(domain.AdID(11), nil)
		m.On("Commit").Return(dbErr)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewImportAdsFunc(m)(ctx, reqs)

		require.ErrorIs(t, err, dbErr)
		assert.Nil(t, got)
	})
}
