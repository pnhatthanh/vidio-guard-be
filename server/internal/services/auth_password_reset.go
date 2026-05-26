package services

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/pnhatthanh/vidio-guard-be/internal/apperror"
	"github.com/pnhatthanh/vidio-guard-be/internal/dto"
	"github.com/pnhatthanh/vidio-guard-be/internal/pkg"
	"github.com/pnhatthanh/vidio-guard-be/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	pwdResetOTPKeyPrefix      = "pwd_reset:otp:"
	pwdResetCooldownKeyPrefix = "pwd_reset:cooldown:"
	pwdResetAttemptsKeyPrefix = "pwd_reset:attempts:"
)

func (s *authService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) (*dto.MessageResponse, error) {
	email := utils.NormalizeEmail(req.Email)
	msg := &dto.MessageResponse{
		Message: "If an account with this email exists, a verification code has been sent",
	}

	cooldownKey := pwdResetCooldownKeyPrefix + email
	if exists, _ := s.cache.IsExist(cooldownKey); exists {
		return msg, nil
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return msg, nil
		}
		return nil, apperror.NewInternalServerError("failed to process request")
	}

	otp, err := generateOTP6()
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to generate verification code")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to store verification code")
	}

	otpKey := pwdResetOTPKeyPrefix + email
	if err := s.cache.Set(otpKey, string(hash), s.pwdReset.OTPTTL); err != nil {
		return nil, apperror.NewInternalServerError("failed to store verification code")
	}
	_ = s.cache.Delete(pwdResetAttemptsKeyPrefix + email)

	if err := s.cache.Set(cooldownKey, "1", s.pwdReset.CooldownTTL); err != nil {
		log.Printf("[password_reset] cooldown for %s: %v", email, err)
	}

	mail := pkg.BuildPasswordResetEmail(
		user.FullName,
		email,
		otp,
		s.pwdReset.ResetPageURL,
		s.pwdReset.OTPTTL,
	)

	if err := s.sendPasswordResetEmail(ctx, user.Email, mail, otp); err != nil {
		log.Printf("[password_reset] send email to %s: %v", email, err)
		return nil, apperror.NewInternalServerError("failed to send verification email")
	}

	return msg, nil
}

func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.MessageResponse, error) {
	email := utils.NormalizeEmail(req.Email)
	otp := strings.TrimSpace(req.OTP)

	otpKey := pwdResetOTPKeyPrefix + email
	storedHash, err := s.cache.Get(otpKey)
	if err != nil || storedHash == "" {
		return nil, apperror.NewBadRequestError("invalid or expired verification code")
	}

	attemptsKey := pwdResetAttemptsKeyPrefix + email
	attempts, _ := s.cache.Get(attemptsKey)
	attemptCount, _ := strconv.Atoi(attempts)
	if attemptCount >= s.pwdReset.MaxAttempts {
		return nil, apperror.NewBadRequestError("too many invalid attempts, request a new code")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(otp)); err != nil {
		n, incrErr := s.cache.Incr(attemptsKey)
		if incrErr == nil && n == 1 {
			_ = s.cache.Expire(attemptsKey, s.pwdReset.AttemptsTTL)
		}
		return nil, apperror.NewBadRequestError("invalid or expired verification code")
	}

	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NewBadRequestError("invalid or expired verification code")
		}
		return nil, apperror.NewInternalServerError("failed to reset password")
	}

	passwordHash, err := utils.HashPassword(req.NewPassword)
	if err != nil {
		return nil, apperror.NewInternalServerError("failed to hash password")
	}

	user.PasswordHash = passwordHash
	if err := s.users.Update(ctx, user); err != nil {
		return nil, apperror.NewInternalServerError("failed to reset password")
	}

	_ = s.cache.Delete(otpKey)
	_ = s.cache.Delete(attemptsKey)
	_ = s.tokens.DeleteByUserID(ctx, user.ID)

	return &dto.MessageResponse{Message: "password reset successfully"}, nil
}

func (s *authService) sendPasswordResetEmail(ctx context.Context, to string, mail pkg.PasswordResetEmail, otp string) error {
	if s.mailer.Enabled() {
		return s.mailer.SendHTML(ctx, to, mail.Subject, mail.HTML, mail.PlainText)
	}
	log.Printf("[password_reset] SMTP not configured — OTP for %s: %s url=%s", to, otp, mail.ResetURL)
	return nil
}

func generateOTP6() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(b[:]) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}
