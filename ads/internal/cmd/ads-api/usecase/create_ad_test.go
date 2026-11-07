package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao/mock"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/usecase"
)

func TestCreateAd(t *testing.T) {
	ctx := context.Background()
	owner := domain.UserID("user-1")

	ucReq := usecase.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    owner,
	}
	daoReq := adsdao.CreateAdRequest{
		Title:      "mountain bike",
		PriceCents: 15000,
		PhotoURL:   "https://example.com/bike.jpg",
		OwnerID:    owner,
	}

	now := time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC)
	// times returns n creation times one hour apart, newest first, like the DAO does.
	times := func(n int) []time.Time {
		ts := make([]time.Time, 0, n)
		for i := range n {
			ts = append(ts, now.Add(-time.Duration(i)*time.Hour))
		}
		return ts
	}

	t.Run("creates the ad when the user is under the quota", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(nil)
		m.On("RecentAdCreations", ctx, owner, 24*time.Hour, 5).Return(times(4), nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(7), nil)
		m.On("Commit").Return(nil)
		m.On("Rollback").Return(nil)

		got, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.NoError(t, err)
		assert.Equal(t, domain.AdID(7), got)
	})

	t.Run("rejects the sixth ad with the time the oldest counted ad leaves the window", func(t *testing.T) {
		m := mock.New(t)
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(nil)
		m.On("RecentAdCreations", ctx, owner, 24*time.Hour, 5).Return(times(5), nil)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		var quotaErr usecase.QuotaExceededError
		require.ErrorAs(t, err, &quotaErr)
		assert.Equal(t, now.Add(-4*time.Hour).Add(24*time.Hour), quotaErr.RetryAt)
		m.AssertNotCalled(t, "CreateAd", ctx, daoReq)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("rolls back when the insert fails", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(nil)
		m.On("RecentAdCreations", ctx, owner, 24*time.Hour, 5).Return(nil, nil)
		m.On("CreateAd", ctx, daoReq).Return(domain.AdID(0), dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.ErrorIs(t, err, dbErr)
		m.AssertNotCalled(t, "Commit")
	})

	t.Run("returns the error when the lock fails", func(t *testing.T) {
		m := mock.New(t)
		dbErr := errors.New("connection reset")
		m.On("Begin", ctx).Return(nil)
		m.On("LockOwner", ctx, owner).Return(dbErr)
		m.On("Rollback").Return(nil)

		_, err := usecase.NewCreateAdFunc(m)(ctx, ucReq)

		require.ErrorIs(t, err, dbErr)
	})
}
