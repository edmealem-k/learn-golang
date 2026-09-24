# 02. Environment Config & RSA Key Generation

In this section, you will learn how asymmetric cryptographic keys (RS256) work, how to generate 2048-bit RSA key pairs using OpenSSL, how to store them safely in environment configuration, and how to use **Viper** to load and parse configuration into strongly-typed Go structs.

---

## 1. Why RS256 Asymmetric Keys?

Most tutorials use **HS256** (Symmetric Signing), where a single secret string is shared between token creation and token verification.

- **Problem with HS256**: Any service that verifies a token must know the secret key. If a verification service is compromised, attackers can forge valid tokens.
- **The RS256 Solution**:
  - **Private Key**: Kept strictly confidential by the auth server to **sign** tokens.
  - **Public Key**: Distributed freely to any service, middleware, or client to **verify** signatures.
  - Even if an attacker steals the public key, they cannot forge tokens.

We will use two key pairs:

1. **Access Token Key Pair**: Signs short-lived access tokens (e.g., 15 minutes).
2. **Refresh Token Key Pair**: Signs longer-lived refresh tokens (e.g., 60 minutes or 7 days).

---

## 2. Generating 2048-bit RSA Key Pairs

Run these OpenSSL commands in your terminal (you can run them inside a temporary directory or directly in your project root):

```bash
# 1. Generate Access Token Private & Public Keys
openssl genrsa -out access_private.pem 2048
openssl rsa -in access_private.pem -pubout -out access_public.pem

# 2. Generate Refresh Token Private & Public Keys
openssl genrsa -out refresh_private.pem 2048
openssl rsa -in refresh_private.pem -pubout -out refresh_public.pem
```

### Converting PEM Keys to Base64 for `.env` Storage

PEM files contain multi-line strings with headers (`-----BEGIN RSA PRIVATE KEY-----`). Storing multi-line strings in `.env` files can cause parsing issues with standard dotenv parsers.

To make them safe for single-line `.env` variables, encode them into **Base64**:

On Linux:

```bash
cat access_private.pem | base64 -w 0
cat access_public.pem | base64 -w 0
cat refresh_private.pem | base64 -w 0
cat refresh_public.pem | base64 -w 0
```

_(On macOS, omit `-w 0`: `cat access_private.pem | base64`)_

---

## 3. Creating `app.env`

Create an `app.env` file in the root of your project:

```env
POSTGRES_HOST=127.0.0.1
POSTGRES_USER=postgres
POSTGRES_PASSWORD=password123
POSTGRES_DB=golang-gorm
POSTGRES_PORT=6500

PORT=8000
CLIENT_ORIGIN=http://localhost:3000

ACCESS_TOKEN_PRIVATE_KEY=<PASTE_BASE64_ACCESS_PRIVATE_KEY_HERE>
ACCESS_TOKEN_PUBLIC_KEY=<PASTE_BASE64_ACCESS_PUBLIC_KEY_HERE>
ACCESS_TOKEN_EXPIRED_IN=15m
ACCESS_TOKEN_MAXAGE=15

REFRESH_TOKEN_PRIVATE_KEY=<PASTE_BASE64_REFRESH_PRIVATE_KEY_HERE>
REFRESH_TOKEN_PUBLIC_KEY=<PASTE_BASE64_REFRESH_PUBLIC_KEY_HERE>
REFRESH_TOKEN_EXPIRED_IN=60m
REFRESH_TOKEN_MAXAGE=60
```

> **Note**: Replace `<PASTE_BASE64_...>` with the Base64 output strings generated from OpenSSL above.

---

## 4. Building the Configuration Loader (`initializers/loadEnv.go`)

Now, create `initializers/loadEnv.go`. We use `spf13/viper` to read the `app.env` file and unmarshal it into a Go struct.

```go
package initializers

import (
	"time"

	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variable.
type Config struct {
	DBHost         string `mapstructure:"POSTGRES_HOST"`
	DBUser         string `mapstructure:"POSTGRES_USER"`
	DBUserPassword string `mapstructure:"POSTGRES_PASSWORD"`
	DBName         string `mapstructure:"POSTGRES_DB"`
	DBPort         string `mapstructure:"POSTGRES_PORT"`
	ServerPort     string `mapstructure:"PORT"`

	ClientOrigin string `mapstructure:"CLIENT_ORIGIN"`

	AccessTokenPrivateKey string        `mapstructure:"ACCESS_TOKEN_PRIVATE_KEY"`
	AccessTokenPublicKey  string        `mapstructure:"ACCESS_TOKEN_PUBLIC_KEY"`
	AccessTokenExpiresIn  time.Duration `mapstructure:"ACCESS_TOKEN_EXPIRED_IN"`
	AccessTokenMaxAge     int           `mapstructure:"ACCESS_TOKEN_MAXAGE"`

	RefreshTokenPrivateKey string        `mapstructure:"REFRESH_TOKEN_PRIVATE_KEY"`
	RefreshTokenPublicKey  string        `mapstructure:"REFRESH_TOKEN_PUBLIC_KEY"`
	RefreshTokenExpiresIn  time.Duration `mapstructure:"REFRESH_TOKEN_EXPIRED_IN"`
	RefreshTokenMaxAge     int           `mapstructure:"REFRESH_TOKEN_MAXAGE"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigType("env")
	viper.SetConfigName("app")

	// Read environment variables that match
	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
```

### Key Highlights:

1. **`mapstructure` tags**: Viper uses `mapstructure` under the hood to map `.env` variable names to Go struct fields.
2. **`time.Duration` parsing**: Viper automatically converts strings like `"15m"` or `"60m"` into `time.Duration` nanosecond values (`15 * time.Minute`), saving manual parsing logic.
3. **Explicit dependency**: Unlike relying on global mutable state or `func init()`, returning `(Config, error)` allows your application entry point to handle failures explicitly.
