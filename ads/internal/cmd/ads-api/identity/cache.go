// Package identity wraps the users client for the read path: nicknames are cached for a short time and the number of calls in flight is capped, so a slow identity service costs one wait per owner instead of one per read.
package identity

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersapi "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
)

const (
	// maxInFlight bounds the calls waiting on the identity service at any time. Beyond it GetUserByID fails immediately with ErrBusy instead of queueing, so a hanging identity service cannot pile up goroutines and connections behind it.
	maxInFlight = 32
	// maxEntries bounds the memory of the cache. When reached, expired entries are dropped; if none are, the new entry is not cached.
	maxEntries = 10_000
)

// ErrBusy is returned when maxInFlight calls are already waiting on the identity service.
var ErrBusy = errors.New("identity service busy")

type entry struct {
	user    usersclient.User
	err     error
	expires time.Time
}

// Cache is a usersclient.UserClient that remembers answers of another one for a while.
type Cache struct {
	inner    usersclient.UserClient
	ttl      time.Duration
	now      func() time.Time
	inFlight chan struct{}

	mu      sync.Mutex
	entries map[model.UserID]entry
}

// NewCache wraps users: GetUserByID answers from memory for ttl after a call, and at most maxInFlight calls wait on users at the same time. Both successes and "user not found" are cached; other errors are not, so a transient failure is retried on the next read.
func NewCache(users usersclient.UserClient, ttl time.Duration) *Cache {
	return &Cache{
		inner:    users,
		ttl:      ttl,
		now:      time.Now,
		inFlight: make(chan struct{}, maxInFlight),
		entries:  make(map[model.UserID]entry),
	}
}

// GetUserByID returns the cached answer for userID, or asks the identity service and caches its answer.
func (c *Cache) GetUserByID(ctx context.Context, userID model.UserID) (usersclient.User, error) {
	if e, ok := c.get(userID); ok {
		return e.user, e.err
	}

	select {
	case c.inFlight <- struct{}{}:
		defer func() { <-c.inFlight }()
	default:
		return usersclient.User{}, ErrBusy
	}

	user, err := c.inner.GetUserByID(ctx, userID)
	if err == nil || errors.Is(err, usersapi.ErrNotFound) {
		c.set(userID, entry{user: user, err: err, expires: c.now().Add(c.ttl)})
	}

	return user, err
}

// ListUsers is not cached: nothing on the read path calls it.
func (c *Cache) ListUsers(ctx context.Context) ([]usersclient.User, error) {
	return c.inner.ListUsers(ctx)
}

func (c *Cache) get(userID model.UserID) (entry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[userID]
	if !ok || !c.now().Before(e.expires) {
		return entry{}, false
	}

	return e, true
}

func (c *Cache) set(userID model.UserID, e entry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= maxEntries {
		now := c.now()
		for id, old := range c.entries {
			if !now.Before(old.expires) {
				delete(c.entries, id)
			}
		}

		if len(c.entries) >= maxEntries {
			return
		}
	}

	c.entries[userID] = e
}
