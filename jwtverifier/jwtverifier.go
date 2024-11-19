package jwtverifier

import (
	"context"
	"time"

	oktajwt "github.com/okta/okta-jwt-verifier-golang/v2"
	"github.com/okta/okta-jwt-verifier-golang/v2/utils"
	inmemorycache "gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/jwtverifier/cache/inmemory"
)

type tokenContextKey int

const key tokenContextKey = iota

type JwtVerifier oktajwt.JwtVerifier

// NewJwtVerifier creates a new JwtVerifier
func NewJwtVerifier(idpAuthorities []string, idpAudience string, cacheExpiration time.Duration) (*JwtVerifier, error) {
	toValidate := map[string]string{}
	toValidate["aud"] = idpAudience

	jwtVerifierSetup := &oktajwt.JwtVerifier{
		Issuers:          idpAuthorities,
		ClaimsToValidate: toValidate,
		Cache: func(lookup func(string) (any, error), timeout, cleanup time.Duration) (utils.Cacher, error) {
			cleanUpInterval := 10 * time.Minute
			return inmemorycache.NewProvider(lookup, cacheExpiration, cleanUpInterval)
		},
	}

	jwtVerifier, err := jwtVerifierSetup.New()
	if err != nil {
		return nil, err
	}

	return (*JwtVerifier)(jwtVerifier), nil
}

// VerifyAccessToken verifies the token against the ID Provider's public cert and then returns the
// claims as a map[string]any
func (jwtVerifier *JwtVerifier) VerifyAccessToken(jwt string) (map[string]any, error) {
	verifier := (*oktajwt.JwtVerifier)(jwtVerifier)

	token, err := verifier.VerifyAccessToken(jwt)
	if err != nil {
		return nil, err
	}

	return token.Claims, err
}

// NewContext returns a new context.Context with the given claims
func NewContext(ctx context.Context, claims map[string]any) context.Context {
	return context.WithValue(ctx, key, claims)
}

// ClaimsFromContext returns the claims from the given context.Context
func ClaimsFromContext(ctx context.Context) map[string]any {
	if claims, ok := ctx.Value(key).(map[string]any); ok {
		return claims
	}

	return nil
}
