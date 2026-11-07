package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/middleware"
	"github.mpi-internal.com/leboncoin/backend-hiring-assessment/common/model"
)

func TestRequireAuth(t *testing.T) {
	const alice = "00000000-0000-0000-0000-000000000001"

	call := func(header string) (*httptest.ResponseRecorder, model.UserID) {
		var seen model.UserID
		h := middleware.RequireAuth(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			seen = middleware.UserIDFromRequest(r)
			rw.WriteHeader(http.StatusNoContent)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			req.Header.Set("Authorization", header)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		return rec, seen
	}

	t.Run("passes a UUID to the handler through the context", func(t *testing.T) {
		rec, seen := call(alice)

		assert.Equal(t, http.StatusNoContent, rec.Code)
		assert.Equal(t, model.UserID(alice), seen)
	})

	t.Run("returns 401 when the header is missing", func(t *testing.T) {
		rec, _ := call("")

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("returns 401 when the header is not a UUID", func(t *testing.T) {
		for _, header := range []string{"not-a-uuid", "user-1", "00000000-0000-0000-0000-00000000000", "Bearer " + alice} {
			rec, _ := call(header)

			assert.Equal(t, http.StatusUnauthorized, rec.Code, header)
		}
	})
}
