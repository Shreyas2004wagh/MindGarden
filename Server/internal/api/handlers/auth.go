package handlers

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

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
	N   string `json:"n"`
	E   string `json:"e"`
}

func authenticatedUserID(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("authorization header required")
	}

	token, ok := strings.CutPrefix(authHeader, "Bearer ")
	if !ok || strings.TrimSpace(token) == "" {
		return "", errors.New("invalid authorization format. Expected: Bearer <token>")
	}

	return verifySupabaseJWT(strings.TrimSpace(token))
}

func verifySupabaseJWT(tokenString string) (string, error) {
	// Parse token header to get kid
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token header: %w", err)
	}

	switch token.Method.Alg() {
	case jwt.SigningMethodHS256.Alg():
		return verifyHS256Token(tokenString)
	case jwt.SigningMethodES256.Alg(), jwt.SigningMethodRS256.Alg():
	default:
		return "", fmt.Errorf("unsupported signing method: %s", token.Method.Alg())
	}

	kid, ok := token.Header["kid"].(string)
	if !ok {
		return "", errors.New("missing kid in token header")
	}

	// Fetch JWKS
	supabaseURL := strings.TrimRight(os.Getenv("SUPABASE_URL"), "/")
	if supabaseURL == "" {
		return "", errors.New("server configuration error: SUPABASE_URL not set")
	}

	jwksURL := supabaseURL + "/auth/v1/.well-known/jwks.json"
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(jwksURL)
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

	publicKey, err := jwkPublicKey(*jwk)
	if err != nil {
		return "", fmt.Errorf("invalid JWK: %w", err)
	}

	// Verify token
	verifiedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != token.Method.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return publicKey, nil
	})
	return subjectFromVerifiedToken(verifiedToken, err)
}

func verifyHS256Token(tokenString string) (string, error) {
	secret := os.Getenv("SUPABASE_JWT_SECRET")
	if secret == "" {
		return "", errors.New("server configuration error: SUPABASE_JWT_SECRET not set")
	}

	verifiedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", t.Method.Alg())
		}
		return []byte(secret), nil
	})
	return subjectFromVerifiedToken(verifiedToken, err)
}

func jwkPublicKey(jwk JWK) (interface{}, error) {
	switch jwk.Kty {
	case "EC":
		if jwk.Alg != "" && jwk.Alg != jwt.SigningMethodES256.Alg() {
			return nil, fmt.Errorf("unsupported EC algorithm: %s", jwk.Alg)
		}
		xBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return nil, err
		}
		yBytes, err := base64.RawURLEncoding.DecodeString(jwk.Y)
		if err != nil {
			return nil, err
		}
		return &ecdsa.PublicKey{
			Curve: ellipticP256(),
			X:     new(big.Int).SetBytes(xBytes),
			Y:     new(big.Int).SetBytes(yBytes),
		}, nil
	case "RSA":
		if jwk.Alg != "" && jwk.Alg != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unsupported RSA algorithm: %s", jwk.Alg)
		}
		nBytes, err := base64.RawURLEncoding.DecodeString(jwk.N)
		if err != nil {
			return nil, err
		}
		eBytes, err := base64.RawURLEncoding.DecodeString(jwk.E)
		if err != nil {
			return nil, err
		}
		e := new(big.Int).SetBytes(eBytes).Int64()
		if e == 0 {
			return nil, errors.New("invalid RSA exponent")
		}
		return &rsa.PublicKey{
			N: new(big.Int).SetBytes(nBytes),
			E: int(e),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported key type: %s", jwk.Kty)
	}
}

func subjectFromVerifiedToken(verifiedToken *jwt.Token, err error) (string, error) {
	if err != nil || verifiedToken == nil || !verifiedToken.Valid {
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
