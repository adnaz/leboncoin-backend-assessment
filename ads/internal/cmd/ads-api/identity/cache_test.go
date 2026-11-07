package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersapi "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
	usersmock "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/mock"
)

func TestCache(t *testing.T) {
	ctx := context.Background()
	alice := usersclient.User{UserID: "user-1", Nickname: "alice"}
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// newCache returns a cache whose clock the test moves by hand.
	newCache := func(users usersclient.UserClient) (*Cache, *time.Time) {
		now := start
		c := NewCache(users, time.Minute)
		c.now = func() time.Time { return now }
		return c, &now
	}

	t.Run("asks the identity service once per user within the TTL", func(t *testing.T) {
		users := usersmock.New(t)
		users.On("GetUserByID", ctx, model.UserID("user-1")).Return(alice, nil).Once()
		c, _ := newCache(users)

		for range 3 {
			got, err := c.GetUserByID(ctx, "user-1")
			require.NoError(t, err)
			assert.Equal(t, alice, got)
		}
	})

	t.Run("asks again once the TTL has passed", func(t *testing.T) {
		users := usersmock.New(t)
		users.On("GetUserByID", ctx, model.UserID("user-1")).Return(alice, nil).Twice()
		c, now := newCache(users)

		_, err := c.GetUserByID(ctx, "user-1")
		require.NoError(t, err)
		*now = start.Add(time.Minute)
		_, err = c.GetUserByID(ctx, "user-1")
		require.NoError(t, err)
	})

	t.Run("caches an unknown user", func(t *testing.T) {
		users := usersmock.New(t)
		users.On("GetUserByID", ctx, model.UserID("ghost")).Return(usersclient.User{}, usersapi.ErrNotFound).Once()
		c, _ := newCache(users)

		for range 2 {
			_, err := c.GetUserByID(ctx, "ghost")
			require.ErrorIs(t, err, usersapi.ErrNotFound)
		}
	})

	t.Run("does not cache a failure of the identity service", func(t *testing.T) {
		users := usersmock.New(t)
		boom := errors.New("identity service timeout")
		users.On("GetUserByID", ctx, model.UserID("user-1")).Return(usersclient.User{}, boom).Once()
		users.On("GetUserByID", ctx, model.UserID("user-1")).Return(alice, nil).Once()
		c, _ := newCache(users)

		_, err := c.GetUserByID(ctx, "user-1")
		require.ErrorIs(t, err, boom)
		got, err := c.GetUserByID(ctx, "user-1")
		require.NoError(t, err)
		assert.Equal(t, alice, got)
	})

	t.Run("fails fast when too many calls are waiting on the identity service", func(t *testing.T) {
		release := make(chan struct{})
		c, _ := newCache(blockingClient{release: release})

		done := make(chan struct{})
		for range maxInFlight {
			go func() {
				_, _ = c.GetUserByID(ctx, "user-1")
				done <- struct{}{}
			}()
		}
		// Every slot is taken once all the goroutines are inside the inner call.
		assert.Eventually(t, func() bool { return len(c.inFlight) == maxInFlight }, time.Second, time.Millisecond)

		_, err := c.GetUserByID(ctx, "user-2")

		require.ErrorIs(t, err, ErrBusy)
		close(release)
		for range maxInFlight {
			<-done
		}
	})
}

// blockingClient holds every GetUserByID call until release is closed.
type blockingClient struct {
	release chan struct{}
}

func (b blockingClient) GetUserByID(_ context.Context, userID model.UserID) (usersclient.User, error) {
	<-b.release
	return usersclient.User{UserID: userID}, nil
}

func (b blockingClient) ListUsers(context.Context) ([]usersclient.User, error) {
	return nil, nil
}
