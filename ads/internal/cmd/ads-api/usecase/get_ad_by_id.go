package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	adsdao "github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/dao"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/ads/internal/cmd/ads-api/domain"
	usersclient "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg"
	usersapi "github.mpi-internal.com/leboncoin/backend-hiring-assessment/users/pkg/api"
)

type (
	// GetAdByIDFunc returns a single ad with its owner's username. It fails with adsdao.ErrNotFound when no ad matches the identifier. A failure of the identity service is not an error: the ad is returned with a nil Username.
	GetAdByIDFunc func(ctx context.Context, id domain.AdID) (AdWithOwner, error)

	// AdWithOwner is an ad and what the identity service knows about its owner. Username is nil when the identity service could not answer or no longer knows the owner.
	AdWithOwner struct {
		Ad       domain.Ad
		Username *string
	}

	getAdByID struct {
		adDAO adsdao.DAO
		users usersclient.UserClient
	}
)

// NewGetAdByIDFunc returns a GetAdByIDFunc backed by the given DAO and identity client.
func NewGetAdByIDFunc(adDAO adsdao.DAO, users usersclient.UserClient) GetAdByIDFunc {
	return getAdByID{adDAO: adDAO, users: users}.getAdByID
}

func (uc getAdByID) getAdByID(ctx context.Context, id domain.AdID) (AdWithOwner, error) {
	found, err := uc.adDAO.GetAdByID(ctx, id)
	if err != nil {
		return AdWithOwner{}, fmt.Errorf("unable to get ad %s: %w", id, err)
	}

	user, err := uc.users.GetUserByID(ctx, found.UserID)
	if err != nil {
		// The read path degrades instead of failing: the ad is served without its owner's username.
		if errors.Is(err, usersapi.ErrNotFound) {
			slog.WarnContext(ctx, "ad owner unknown to the identity service", "ad_id", id, "user_id", found.UserID)
		} else {
			slog.WarnContext(ctx, "unable to get the ad owner", "ad_id", id, "error", err)
		}

		return AdWithOwner{Ad: found}, nil
	}

	return AdWithOwner{Ad: found, Username: &user.Nickname}, nil
}
