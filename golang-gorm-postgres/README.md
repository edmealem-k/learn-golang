# Golang + GORM + PostgreSQL: RESTful API with RS256 JWT & HttpOnly Cookies

Welcome to the **Golang + GORM + PostgreSQL** RESTful API project!

This project focuses on enterprise-level authentication and modern REST API patterns in Go:

- **Asymmetric RS256 JWT**: Using 2048-bit RSA Private and Public key pairs (the same method used by Auth0, Okta, and OAuth2/OIDC).
- **HttpOnly Cookie Delivery**: Delivering Access and Refresh tokens via secure browser cookies to prevent Cross-Site Scripting (XSS) attacks, while also supporting standard `Authorization: Bearer <token>` headers for mobile clients.
- **Environment Management with Viper**: Loading and validating configuration from `.env` files with automatic duration parsing.
- **Data Modeling with UUIDs**: Native PostgreSQL UUID primary keys.
- **Protected Post CRUD**: Managing posts linked to authenticated users.

---

## 📚 Step-by-Step Learning Guide

Inside the [`docs/`](./docs) folder, you will find detailed, blog-post-style guides designed for you to code every single layer step-by-step with full explanations and code snippets:

| Guide                                                                                             | Topic & What You Will Build                                                                                                         |
| ------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| [**01. Project Setup & Docker**](./docs/01-project-setup-and-docker.md)                           | Initialize Go module, create modular project structure, install dependencies, and configure PostgreSQL with Docker Compose.         |
| [**02. Environment Config & RSA Keys**](./docs/02-environment-config-and-rsa-keys.md)             | Generate 2048-bit RSA Private/Public keys with OpenSSL, encode in Base64, and implement the Viper configuration loader (`app.env`). |
| [**03. Database Connection & Models**](./docs/03-database-connection-and-models.md)               | Connect to PostgreSQL with GORM, enable the `uuid-ossp` extension, and define `User` and `Post` models with UUID primary keys.      |
| [**04. Password Hashing & RS256 JWT**](./docs/04-password-hashing-and-rs256-jwt.md)               | Implement Bcrypt password hashing and RS256 token creation and validation using modern `golang-jwt/jwt/v5`.                         |
| [**05. Auth Controllers & Cookie Delivery**](./docs/05-authentication-controllers-and-cookies.md) | Build `SignUpUser`, `SignInUser`, `RefreshAccessToken`, and `LogoutUser` with secure `HttpOnly` cookie delivery.                    |
| [**06. Middleware Guard (DeserializeUser)**](./docs/06-middleware-deserialize-user.md)            | Create authentication middleware that extracts tokens from either cookies or `Bearer` headers and validates with the Public Key.    |
| [**07. Post CRUD Controller**](./docs/07-post-crud-controller.md)                                 | Build `CreatePost`, `FindPosts` (with limit/offset pagination), `FindPostById`, `UpdatePost`, and `DeletePost`.                     |
| [**08. Routes & Application Entry Point**](./docs/08-routes-and-main-application.md)              | Group routes into controller structs, configure CORS middleware, and assemble everything in `main.go`.                              |

---

## 🛠️ Tech Stack & Packages

- **Web Framework**: [Gin Gonic](https://github.com/gin-gonic/gin) (`github.com/gin-gonic/gin`)
- **ORM**: [GORM](https://gorm.io) (`gorm.io/gorm`) & PostgreSQL Driver (`gorm.io/driver/postgres`)
- **Configuration**: [Viper](https://github.com/spf13/viper) (`github.com/spf13/viper`)
- **Authentication**: [golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt) (`github.com/golang-jwt/jwt/v5`)
- **Password Hashing**: [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt)
- **UUIDs**: [google/uuid](https://github.com/google/uuid) (`github.com/google/uuid`)
- **CORS**: [gin-contrib/cors](https://github.com/gin-contrib/cors) (`github.com/gin-contrib/cors`)

---

## 🚀 Quick Start (When Ready)

1. Open [docs/01-project-setup-and-docker.md](./docs/01-project-setup-and-docker.md) to begin step 1!
2. Follow each guide sequentially to write and understand your code.
