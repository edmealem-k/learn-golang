# 08. Complete Authentication Implementation Guide

This guide provides the complete, unabridged code for the entire Authentication subsystem, including:

1. **User & Token Repositories** (`internal/repository`)
2. **Email Service** for sending password reset links (`internal/service`)
3. **Authentication Service** (`internal/service`)
4. **Authentication Handler** (`internal/handler`)

---

## 1. User & Token Repositories

### `internal/repository/user_repository.go`

```go
package repository

import (
	"context"
	"go-task-manager/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error
}

type postgresUserRepository struct {
	db *gorm.DB
}

func NewPostgresUserRepository(db *gorm.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *postgresUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, newPasswordHash string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Update("password_hash", newPasswordHash).Error
}
```

---

### `internal/repository/token_repository.go`

```go
package repository

import (
	"context"
	"time"

	"go-task-manager/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TokenRepository interface {
	// Refresh Tokens
	CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error
	FindRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error

	// Password Reset Tokens
	CreateResetToken(ctx context.Context, token *models.PasswordResetToken) error
	FindValidResetToken(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)
	MarkResetTokenAsUsed(ctx context.Context, tokenHash string) error
}

type postgresTokenRepository struct {
	db *gorm.DB
}

func NewPostgresTokenRepository(db *gorm.DB) TokenRepository {
	return &postgresTokenRepository{db: db}
}

func (r *postgresTokenRepository) CreateRefreshToken(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *postgresTokenRepository) FindRefreshToken(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *postgresTokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("token_hash = ?", tokenHash).
		Update("revoked_at", &now).Error
}

func (r *postgresTokenRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", &now).Error
}

func (r *postgresTokenRepository) CreateResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *postgresTokenRepository) FindValidResetToken(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	var token models.PasswordResetToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *postgresTokenRepository) MarkResetTokenAsUsed(ctx context.Context, tokenHash string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.PasswordResetToken{}).
		Where("token_hash = ?", tokenHash).
		Update("used_at", &now).Error
}
```

---

## 2. Email Service (`internal/service/email_service.go`)

Decoupling email sending behind an interface allows you to use mock emails in development / unit tests and real SMTP in staging or production.

```go
package service

import (
	"fmt"
	"log"
	"net/smtp"

	"go-task-manager/internal/config"
)

type EmailService interface {
	SendPasswordResetEmail(toEmail, resetToken string) error
}

type smtpEmailService struct {
	cfg *config.Config
}

func NewSMTPEmailService(cfg *config.Config) EmailService {
	return &smtpEmailService{cfg: cfg}
}

func (s *smtpEmailService) SendPasswordResetEmail(toEmail, resetToken string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.FrontendURL, resetToken)

	// If SMTP is not configured, print to console for development convenience
	if s.cfg.SMTPUser == "" {
		log.Printf("\n[DEVELOPMENT EMAIL MOCK]\nTo: %s\nSubject: Password Reset Request\nLink: %s\n\n", toEmail, resetURL)
		return nil
	}

	subject := "Subject: Reset Your Password - Task Manager\n"
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<h2>Password Reset Request</h2>
		<p>You requested to reset your password. Click the link below to set a new password:</p>
		<p><a href="%s" style="background:#2563EB;color:#fff;padding:10px 20px;text-decoration:none;border-radius:5px;">Reset Password</a></p>
		<p>This link will expire in 15 minutes. If you did not request this, please ignore this email.</p>
	`, resetURL)

	msg := []byte(subject + mime + body)
	auth := smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPassword, s.cfg.SMTPHost)
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)

	return smtp.SendMail(addr, auth, s.cfg.FromEmail, []string{toEmail}, msg)
}
```

---

## 3. Auth Service (`internal/service/auth_service.go`)

This service orchestrates password verification, token generation, and database updates.

```go
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go-task-manager/internal/config"
	"go-task-manager/internal/models"
	"go-task-manager/internal/repository"
	"go-task-manager/pkg/utils"

	"gorm.io/gorm"
)

var (
	ErrUserAlreadyExists = errors.New("a user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken      = errors.New("invalid or expired token")
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // Seconds
}

type AuthService interface {
	SignUp(ctx context.Context, name, email, password string) (*models.User, error)
	Login(ctx context.Context, email, password, userAgent, clientIP string) (*TokenResponse, error)
	RefreshToken(ctx context.Context, refreshTokenString string) (*TokenResponse, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, rawToken, newPassword string) error
}

type authService struct {
	userRepo  repository.UserRepository
	tokenRepo repository.TokenRepository
	emailSvc  EmailService
	cfg       *config.Config
	db        *gorm.DB // Needed for multi-step ACID transactions
}

func NewAuthService(
	userRepo repository.UserRepository,
	tokenRepo repository.TokenRepository,
	emailSvc EmailService,
	cfg *config.Config,
	db *gorm.DB,
) AuthService {
	return &authService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		emailSvc:  emailSvc,
		cfg:       cfg,
		db:        db,
	}
}

func (s *authService) SignUp(ctx context.Context, name, email, password string) (*models.User, error) {
	// 1. Verify email does not already exist
	existing, _ := s.userRepo.FindByEmail(ctx, email)
	if existing != nil {
		return nil, ErrUserAlreadyExists
	}

	// 2. Hash password with bcrypt
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: hashedPassword,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *authService) Login(ctx context.Context, email, password, userAgent, clientIP string) (*TokenResponse, error) {
	// 1. Find user by email
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// 2. Verify password
	if !utils.CheckPasswordHash(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	// 3. Generate Access JWT
	accessToken, err := utils.GenerateToken(user.ID, user.Email, s.cfg.JWTSecret, s.cfg.JWTAccessExpiration)
	if err != nil {
		return nil, err
	}

	// 4. Generate Random Refresh Token
	rawRefreshToken, err := utils.GenerateRandomHex(32)
	if err != nil {
		return nil, err
	}

	// 5. Store hashed refresh token in database
	tokenModel := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: utils.HashToken(rawRefreshToken),
		UserAgent: userAgent,
		ClientIP:  clientIP,
		ExpiresAt: time.Now().Add(s.cfg.JWTRefreshExpiration),
	}
	if err := s.tokenRepo.CreateRefreshToken(ctx, tokenModel); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken, // Send raw token only to client
		ExpiresIn:    int64(s.cfg.JWTAccessExpiration.Seconds()),
	}, nil
}

func (s *authService) RefreshToken(ctx context.Context, rawRefreshToken string) (*TokenResponse, error) {
	hashed := utils.HashToken(rawRefreshToken)

	// 1. Verify token exists, is not revoked, and is not expired
	tokenRecord, err := s.tokenRepo.FindRefreshToken(ctx, hashed)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 2. Fetch user
	user, err := s.userRepo.FindByID(ctx, tokenRecord.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 3. Issue new Access Token
	newAccessToken, err := utils.GenerateToken(user.ID, user.Email, s.cfg.JWTSecret, s.cfg.JWTAccessExpiration)
	if err != nil {
		return nil, err
	}

	// 4. Token Rotation: Revoke old refresh token and issue a fresh one
	_ = s.tokenRepo.RevokeRefreshToken(ctx, hashed)

	newRawRefreshToken, _ := utils.GenerateRandomHex(32)
	newRecord := &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: utils.HashToken(newRawRefreshToken),
		UserAgent: tokenRecord.UserAgent,
		ClientIP:  tokenRecord.ClientIP,
		ExpiresAt: time.Now().Add(s.cfg.JWTRefreshExpiration),
	}
	_ = s.tokenRepo.CreateRefreshToken(ctx, newRecord)

	return &TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRawRefreshToken,
		ExpiresIn:    int64(s.cfg.JWTAccessExpiration.Seconds()),
	}, nil
}

func (s *authService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		// Silent success to prevent email enumeration attack!
		return nil
	}

	// Generate 32-byte cryptographically secure random token
	rawToken, err := utils.GenerateRandomHex(32)
	if err != nil {
		return err
	}

	resetRecord := &models.PasswordResetToken{
		UserID:    user.ID,
		TokenHash: utils.HashToken(rawToken),
		ExpiresAt: time.Now().Add(15 * time.Minute), // 15 minute lifespan
	}

	if err := s.tokenRepo.CreateResetToken(ctx, resetRecord); err != nil {
		return err
	}

	// Send reset email asynchronously (don't block the HTTP response)
	go func() {
		_ = s.emailSvc.SendPasswordResetEmail(user.Email, rawToken)
	}()

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	hashedToken := utils.HashToken(rawToken)

	// 1. Verify token
	tokenRecord, err := s.tokenRepo.FindValidResetToken(ctx, hashedToken)
	if err != nil {
		return ErrInvalidToken
	}

	// 2. Hash new password
	newHash, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 3. Execute in an ACID Transaction: update password, consume token, revoke sessions
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// A. Update password
		if err := tx.Model(&models.User{}).Where("id = ?", tokenRecord.UserID).Update("password_hash", newHash).Error; err != nil {
			return err
		}

		// B. Mark reset token as used
		now := time.Now()
		if err := tx.Model(&models.PasswordResetToken{}).Where("id = ?", tokenRecord.ID).Update("used_at", &now).Error; err != nil {
			return err
		}

		// C. Invalidate all active refresh tokens (forces re-login across all user devices)
		if err := tx.Model(&models.RefreshToken{}).Where("user_id = ? AND revoked_at IS NULL", tokenRecord.UserID).Update("revoked_at", &now).Error; err != nil {
			return err
		}

		return nil
	})
}
```

---

## 4. Auth Handler (`internal/handler/auth_handler.go`)

```go
package handler

import (
	"errors"
	"net/http"

	"go-task-manager/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) SignUp(c *gin.Context) {
	var req SignUpRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.SignUp(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists) {
			SendError(c, http.StatusConflict, err.Error())
			return
		}
		SendError(c, http.StatusInternalServerError, "Failed to register user")
		return
	}

	SendSuccess(c, http.StatusCreated, "User registered successfully", gin.H{
		"id":    user.ID,
		"name":  user.Name,
		"email": user.Email,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	userAgent := c.GetHeader("User-Agent")
	clientIP := c.ClientIP()

	tokens, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, userAgent, clientIP)
	if err != nil {
		SendError(c, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	SendSuccess(c, http.StatusOK, "Login successful", tokens)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		SendError(c, http.StatusBadRequest, "refresh_token is required")
		return
	}

	tokens, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		SendError(c, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	SendSuccess(c, http.StatusOK, "Token refreshed", tokens)
}

func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	_ = h.authService.ForgotPassword(c.Request.Context(), req.Email)

	// Always return 200 OK regardless of whether email exists
	SendSuccess(c, http.StatusOK, "If your email is registered, you will receive a password reset link shortly.", nil)
}

func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		SendError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		SendError(c, http.StatusBadRequest, "Invalid or expired reset token")
		return
	}

	SendSuccess(c, http.StatusOK, "Password reset successfully. You may now log in with your new password.", nil)
}
```

---

With this guide, you have the exact, full code needed to implement complete, production-grade authentication with session revocation, token rotation, and password reset flows!
