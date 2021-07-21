package server

import (
	"context"
	"io"
	"net/http"
	"regexp"

	"github.com/dgrijalva/jwt-go/v4"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	iam "github.com/netsoc/iam/client"
	"github.com/netsoc/webspaced/pkg/util"
	log "github.com/sirupsen/logrus"
)

type key int

const (
	keyToken key = iota
	keyClaims
	keyUser
	keyWebspace
)

var tokenHeaderRegex = regexp.MustCompile(`^Bearer\s+(\S+)$`)

// UserClaims represents claims in an auth JWT
type UserClaims struct {
	jwt.StandardClaims
	IsAdmin bool `json:"is_admin"`
	Version uint `json:"version"`
}

// Extract the (unverified!) claims
func claimsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		matches := tokenHeaderRegex.FindStringSubmatch(r.Header.Get("Authorization"))
		if len(matches) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), keyToken, matches[1]))

		t, _, err := jwt.NewParser().ParseUnverified(matches[1], &UserClaims{})
		if err != nil {
			util.JSONErrResponse(w, err, http.StatusUnauthorized)
			return
		}

		claims := t.Claims.(*UserClaims)
		r = r.WithContext(context.WithValue(r.Context(), keyClaims, claims))

		next.ServeHTTP(w, r)
	})
}

func writeAccessLog(w io.Writer, params handlers.LogFormatterParams) {
	var uid string
	c := params.Request.Context().Value(keyClaims)
	if c != nil {
		uid = c.(*UserClaims).Subject
	}

	level := log.DebugLevel
	if params.URL.Path == "/health" {
		level = log.TraceLevel
	}
	log.StandardLogger().
		WithFields(log.Fields{
			"uid":     uid,
			"agent":   params.Request.UserAgent(),
			"status":  params.StatusCode,
			"resSize": params.Size,
		}).
		Logf(level, "%v %v", params.Request.Method, params.URL.RequestURI())
}

// authMiddleware is a middleware for validating an IAM token and retrieving the user's details
type authMiddleware struct {
	IAM *iam.APIClient

	NeedAdmin bool
}

func (m *authMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(keyToken)
		if v == nil {
			util.JSONErrResponse(w, util.ErrTokenRequired, 0)
			return
		}
		t := v.(string)

		ctx := context.WithValue(r.Context(), iam.ContextAccessToken, t)
		if _, err := m.IAM.UsersApi.ValidateToken(ctx); err != nil {
			util.JSONErrResponse(w, err, http.StatusForbidden)
			return
		}

		// If the token is valid, the claim that the user is an admin is also valid (changing admin status rolls
		// the token version)
		c := r.Context().Value(keyClaims).(*UserClaims)
		if m.NeedAdmin && !c.IsAdmin {
			util.JSONErrResponse(w, util.ErrAdminRequired, 0)
			return
		}

		username, ok := mux.Vars(r)["username"]
		if !ok {
			username = "self"
		}

		u, _, err := m.IAM.UsersApi.GetUser(ctx, username)
		if err != nil {
			util.JSONErrResponse(w, err, 0)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), keyUser, &u))
		next.ServeHTTP(w, r)
	})
}

func (s *Server) getWebspaceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(keyUser).(*iam.User)
		ws, err := s.Webspaces.Get(int(user.Id), user)
		if err != nil {
			util.JSONErrResponse(w, err, 0)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), keyWebspace, ws))
		next.ServeHTTP(w, r)
	})
}
