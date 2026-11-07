package dao

import (
	"context"
	"time"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
)

// Store reads and writes ads. It is implemented both by the DAO, where every call is its own transaction, and by a Tx opened from it.
type Store interface {
	// CreateAd inserts an ad and returns its generated identifier.
	CreateAd(ctx context.Context, req CreateAdRequest) (domain.AdID, error)
	// GetAdByID returns a single ad, or ErrNotFound.
	GetAdByID(ctx context.Context, id domain.AdID) (domain.Ad, error)
	// DeleteAd deletes the ad only if it belongs to ownerID. It returns ErrNotFound both when no ad has this id and when the ad belongs to someone else, so a caller cannot tell the two apart.
	DeleteAd(ctx context.Context, id domain.AdID, ownerID domain.UserID) error
	// SearchAds returns all ads matching the filters.
	SearchAds(ctx context.Context, req SearchAdsRequest) ([]domain.Ad, error)
	// RecentAdCreations returns the creation times of the newest ads of ownerID created within the last window, newest first, at most limit of them. The database clock defines "now", the same clock that fills created_at.
	RecentAdCreations(ctx context.Context, ownerID domain.UserID, window time.Duration, limit int) ([]time.Time, error)
}

// DAO stores and retrieves ads.
type DAO interface {
	Store
	// Begin opens a transaction. Every call made on the returned Tx is part of it, and the caller must end it with Commit or Rollback.
	Begin(ctx context.Context) (Tx, error)
}

// Tx is a Store scoped to an opened transaction.
type Tx interface {
	Store
	// Commit applies every change made through the transaction.
	Commit() error
	// Rollback discards every change made through the transaction. Calling it on an already committed transaction does nothing, so `defer func() { _ = tx.Rollback() }()` is a safe guard.
	Rollback() error
	// LockOwner serialises the transactions that call it for the same owner until this one ends. Callers must count and insert in separate statements after it: under READ COMMITTED every statement takes its own snapshot, so a statement started after the lock is granted sees what the previous holder committed. Folding the lock and the count into one statement brings the race back.
	LockOwner(ctx context.Context, ownerID domain.UserID) error
}

// SearchAdsRequest describes an ad search. Nil pointers and empty strings mean "no filter on this field".
type SearchAdsRequest struct {
	OwnerID       domain.UserID
	MinPriceCents int64
	MaxPriceCents int64
	Title         string
}

// CreateAdRequest carries the data needed to create an ad.
type CreateAdRequest struct {
	Title      string
	PriceCents int64
	PhotoURL   string
	OwnerID    domain.UserID
}

const (
	// ErrNotFound is returned when no ad matches the requested identifier.
	ErrNotFound sentinelError = "ad not found"
	// ErrUnknownOwner is returned when an ad references a user that does not exist.
	ErrUnknownOwner sentinelError = "unknown owner"
)

type sentinelError string

// Error implements the error interface.
func (e sentinelError) Error() string { return string(e) }
