package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)


type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	Alg string `json:"alg"`
	X   string `json:"x"`
	Y   string `json:"y"`
}

func verifySupabaseJWT(tokenString string) (string, error) {
	// Parse token header to get kid
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token header: %w", err)
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return "", errors.New("missing kid in token header")
	}

	// Fetch JWKS
	jwksURL := os.Getenv("SUPABASE_URL") + "/auth/v1/.well-known/jwks.json"
	resp, err := http.Get(jwksURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("JWKS fetch failed with status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return "", fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// Find matching key
	var jwk *JWK
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			jwk = &key
			break
		}
	}
	if jwk == nil {
		return "", errors.New("matching JWK not found")
	}

	if jwk.Kty != "EC" || jwk.Alg != "ES256" {
		return "", errors.New("unsupported key type or algorithm")
	}

	// Decode EC public key
	xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
	if err != nil {
		return "", err
	}
	yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
	if err != nil {
		return "", err
	}

	publicKey := &ecdsa.PublicKey{
		Curve: ellipticP256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}

	// Verify token
	verifiedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodES256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return publicKey, nil
	})
	if err != nil || !verifiedToken.Valid {
		return "", errors.New("token verification failed")
	}

	claims, ok := verifiedToken.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid token claims")
	}

	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("missing sub claim")
	}

	return sub, nil
}

// Explicit curve reference (avoid nil curve bug)
func ellipticP256() elliptic.Curve {
	return elliptic.P256()
}
