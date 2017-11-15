package authn

import (
	"errors"
	jwt "gopkg.in/square/go-jose.v2/jwt"
	"net/url"
	"time"
)

type idTokenVerifier struct {
	config          Config
	kchain          jwkProvider
	init_time       time.Time
	conf_issuer_url *url.URL
}

func newIdTokenVerifier(config Config, kchain jwkProvider) (*idTokenVerifier, error) {
	conf_issuer, err := url.Parse(config.Issuer)
	if err != nil {
		return nil, err
	}

	return &idTokenVerifier{
		config:          config,
		kchain:          kchain,
		init_time:       time.Now(),
		conf_issuer_url: conf_issuer,
	}, nil
}

func (verifier *idTokenVerifier) GetVerifiedClaims(id_token string) (*jwt.Claims, error) {
	var err error

	claims, err := verifier.get_claims(id_token)
	if err != nil {
		return nil, err
	}

	err = verifier.verify(claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (verifier *idTokenVerifier) get_claims(id_token string) (*jwt.Claims, error) {
	var err error

	id_jwt, err := jwt.ParseSigned(id_token)
	if err != nil {
		return nil, err
	}

	headers := id_jwt.Headers
	if len(headers) != 1 {
		return nil, errors.New("Multi-signature JWT not supported or missing headers information")
	}
	key_id := headers[0].KeyID
	keys, err := verifier.kchain.Key(key_id)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, errors.New("No keys found")
	}
	key := keys[0]

	claims := &jwt.Claims{}
	err = id_jwt.Claims(key, claims)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (verifier *idTokenVerifier) verify(claims *jwt.Claims) error {
	var err error

	// Standard validator uses exact matching instead of URL matching
	err = verifier.verify_token_from_us(claims)
	if err != nil {
		return err
	}

	// Validate rest of the claims
	err = claims.Validate(jwt.Expected{
		Time:     verifier.init_time,
		Audience: jwt.Audience{verifier.config.Audience},
	})
	if err != nil {
		return err
	}

	return nil
}

func (verifier *idTokenVerifier) verify_token_from_us(claims *jwt.Claims) error {
	token_issuer, err := url.Parse(claims.Issuer)
	if err != nil {
		return err
	}
	if verifier.conf_issuer_url.String() != token_issuer.String() {
		return jwt.ErrInvalidIssuer
	}
	return nil
}
