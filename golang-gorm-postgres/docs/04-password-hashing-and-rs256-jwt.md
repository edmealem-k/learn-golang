# 04. Password Hashing & Asymmetric RS256 JWT Utilities

In this section, we implement cryptographic security utilities:

1. **Bcrypt** for hashing and verifying user passwords.
2. **RS256 JWT** for creating and validating asymmetric access and refresh tokens using `golang-jwt/jwt/v5`.

---

## 1. Password Hashing with Bcrypt (`utils/password.go`)

Never store plaintext passwords in a database! **Bcrypt** is a standard, slow password hashing algorithm designed to resist brute-force and dictionary attacks by incorporating a salt and an adaptive work factor.

Create `utils/password.go`:

```go
package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a raw plaintext password using bcrypt with the default cost (10).
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("could not hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// VerifyPassword compares a stored bcrypt hash against a candidate plaintext password.
// Returns nil if the passwords match, or bcrypt.ErrMismatchedHashAndPassword on mismatch.
func VerifyPassword(hashedPassword string, candidatePassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(candidatePassword))
}
```

### Why `bcrypt.CompareHashAndPassword`?

- **Timing attack prevention**: Comparing password hashes with standard string comparison (`==`) leaks timing information. `bcrypt.CompareHashAndPassword` runs in constant time.
- **Embedded Salt**: The salt is automatically embedded in the generated hash string, so you don't need a separate database column for the salt.

---

## 2. Asymmetric RS256 Token Utilities (`utils/token.go`)

We will use **`github.com/golang-jwt/jwt/v5`** to handle RS256 asymmetric signing and verification.

Create `utils/token.go`:

```go
package utils

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CreateToken generates a signed RS256 JWT containing the given payload and TTL.
func CreateToken(ttl time.Duration, payload interface{}, privateKey string) (string, error) {
	// 1. Decode the base64-encoded PEM private key
	decodedPrivateKey, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		return "", fmt.Errorf("could not decode private key: %w", err)
	}

	// 2. Parse the RSA private key from PEM bytes
	key, err := jwt.ParseRSAPrivateKeyFromPEM(decodedPrivateKey)
	if err != nil {
		return "", fmt.Errorf("could not parse private key: %w", err)
	}

	now := time.Now().UTC()

	// 3. Set standard JWT claims
	claims := jwt.MapClaims{
		"sub": payload,                                 // Subject (User ID)
		"exp": now.Add(ttl).Unix(),                     // Expiration time
		"iat": now.Unix(),                              // Issued at
		"nbf": now.Unix(),                              // Not valid before
	}

	// 4. Create and sign the token using RS256
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken verifies the authenticity and expiration of an RS256 token using the public key.
func ValidateToken(tokenString string, publicKey string) (interface{}, error) {
	// 1. Decode the base64-encoded PEM public key
	decodedPublicKey, err := base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		return nil, fmt.Errorf("could not decode public key: %w", err)
	}

	// 2. Parse the RSA public key from PEM bytes
	key, err := jwt.ParseRSAPublicKeyFromPEM(decodedPublicKey)
	if err != nil {
		return nil, fmt.Errorf("could not parse public key: %w", err)
	}

	// 3. Parse and validate the token signature and claims
	parsedToken, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		// Ensure the signing algorithm is RSA
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return key, nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// 4. Extract claims and return the subject (payload)
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims["sub"], nil
}
```

---

## 3. How the Token Flow Works Under the Hood

```
+-----------------------------------------------------------+
|                      Token Issuance                       |
| User ID (uuid)                                            |
|       +                                                   |
| Expiration Time (15m)  ==> RS256 Signing ==> Access Token |
|       +                    (Private Key)                  |
| Private Key (Secret)                                      |
+-----------------------------------------------------------+

+-----------------------------------------------------------+
|                    Token Verification                     |
| Access Token                                              |
|       +                ==> RS256 Verification ==> User ID |
| Public Key (Open)          (Public Key)                   |
+-----------------------------------------------------------+
```

1. **`jwt.SigningMethodRS256`**: Tells the library to use SHA-256 with RSA encryption.
2. **`token.Claims.(jwt.MapClaims)`**: Extracts the claims dictionary. `parsedToken.Valid` automatically verifies whether `exp` (expiration) has passed or if `nbf` (not before) is in the future.
3. If an attacker tampers with the user ID inside the token payload, the cryptographic signature check fails against the public key, immediately rejecting the request.
