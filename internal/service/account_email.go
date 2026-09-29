package service

import (
	"context"
	"errors"
	"net/url"
	"time"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/mail"
	"warta/internal/model"
	"warta/internal/repository"
	"warta/internal/validation"
)

const (
	verifyEmailTTL   = 24 * time.Hour
	resetPasswordTTL = time.Hour
	// emailCooldown adalah jeda minimal antar email dengan tujuan yang sama
	// ke satu akun, supaya endpoint tidak dipakai membanjiri kotak masuk.
	emailCooldown = time.Minute
)

var (
	errBadVerifyToken = apperr.Validation(map[string]string{"token": "tautan verifikasi tidak valid atau sudah kedaluwarsa"})
	errBadResetToken  = apperr.Validation(map[string]string{"token": "tautan reset password tidak valid atau sudah kedaluwarsa"})
)

func (s *authService) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (dto.UserResponse, error) {
	if req.Token == "" {
		return dto.UserResponse{}, apperr.Validation(map[string]string{"token": "token wajib diisi"})
	}

	token, err := s.useToken(ctx, model.TokenVerifyEmail, req.Token)
	if err != nil {
		if errors.Is(err, errTokenInvalid) {
			return dto.UserResponse{}, errBadVerifyToken
		}
		return dto.UserResponse{}, apperr.Internal(err)
	}

	if err := s.users.MarkEmailVerified(ctx, token.UserID); err != nil {
		return dto.UserResponse{}, apperr.Internal(err)
	}
	user, err := s.users.FindByID(ctx, token.UserID)
	if err != nil {
		return dto.UserResponse{}, apperr.Internal(err)
	}
	return dto.NewUserResponse(user), nil
}

func (s *authService) ResendVerification(ctx context.Context, actor auth.Actor) error {
	user, err := s.current(ctx, actor)
	if err != nil {
		return err
	}
	if user.EmailVerifiedAt != nil {
		return apperr.Conflict("email sudah terverifikasi")
	}

	recent, err := s.userTokens.IssuedSince(ctx, user.ID, model.TokenVerifyEmail, s.now().Add(-emailCooldown))
	if err != nil {
		return apperr.Internal(err)
	}
	if recent {
		return apperr.TooManyRequests("email verifikasi baru saja dikirim, tunggu sebentar sebelum meminta lagi")
	}

	if err := s.sendVerification(ctx, user); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *authService) ForgotPassword(ctx context.Context, req dto.ForgotPasswordRequest) error {
	req.Normalize()
	if problems := validation.ValidateForgotPassword(req); len(problems) > 0 {
		return apperr.Validation(problems)
	}

	user, err := s.users.FindByEmail(ctx, req.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return apperr.Internal(err)
	}

	// Permintaan beruntun cukup dilayani sekali; jawabannya tetap sama
	// supaya tidak membocorkan apa pun.
	recent, err := s.userTokens.IssuedSince(ctx, user.ID, model.TokenResetPassword, s.now().Add(-emailCooldown))
	if err != nil {
		return apperr.Internal(err)
	}
	if recent {
		return nil
	}

	link, err := s.newLink(ctx, user.ID, model.TokenResetPassword, resetPasswordTTL, "/reset-password")
	if err != nil {
		return apperr.Internal(err)
	}
	if err := s.mailer.Send(ctx, mail.ResetPassword(user.Email, user.Name, link, "1 jam")); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

// ResetPassword mengganti password lewat tautan email, lalu mencabut semua
// sesi. Karena tautan hanya sampai ke pemilik kotak masuk, emailnya sekaligus
// dianggap terverifikasi.
func (s *authService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) error {
	if problems := validation.ValidateResetPassword(req); len(problems) > 0 {
		return apperr.Validation(problems)
	}

	// Hash dibuat sebelum token dipakai supaya kegagalan bcrypt tidak
	// menghabiskan tautan.
	hash, err := s.hasher.Hash(req.NewPassword)
	if err != nil {
		return apperr.Internal(err)
	}

	token, err := s.useToken(ctx, model.TokenResetPassword, req.Token)
	if err != nil {
		if errors.Is(err, errTokenInvalid) {
			return errBadResetToken
		}
		return apperr.Internal(err)
	}

	if err := s.users.UpdatePassword(ctx, token.UserID, hash); err != nil {
		return apperr.Internal(err)
	}
	if err := s.tokens.RevokeAllForUser(ctx, token.UserID); err != nil {
		return apperr.Internal(err)
	}
	if err := s.users.MarkEmailVerified(ctx, token.UserID); err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *authService) sendVerification(ctx context.Context, user model.User) error {
	link, err := s.newLink(ctx, user.ID, model.TokenVerifyEmail, verifyEmailTTL, "/verify-email")
	if err != nil {
		return err
	}
	return s.mailer.Send(ctx, mail.VerifyEmail(user.Email, user.Name, link, "24 jam"))
}

// newLink membuat token sekali pakai (yang lama untuk tujuan sama ikut
// dicabut) dan mengembalikan tautan frontend yang memuatnya.
func (s *authService) newLink(ctx context.Context, userID int64, purpose string, ttl time.Duration, path string) (string, error) {
	plain, hash, err := auth.NewRefreshToken()
	if err != nil {
		return "", err
	}

	token := model.UserToken{UserID: userID, Purpose: purpose, TokenHash: hash, ExpiresAt: s.now().Add(ttl)}
	if err := s.userTokens.Replace(ctx, &token); err != nil {
		return "", err
	}
	return s.appURL + path + "?token=" + url.QueryEscape(plain), nil
}

var errTokenInvalid = errors.New("token tidak valid")

// useToken memeriksa token dan menandainya terpakai. Hanya satu permintaan
// yang bisa memakai token yang sama walau datang bersamaan.
func (s *authService) useToken(ctx context.Context, purpose, plain string) (model.UserToken, error) {
	token, err := s.userTokens.FindByHash(ctx, purpose, auth.HashRefreshToken(plain))
	if errors.Is(err, repository.ErrNotFound) {
		return model.UserToken{}, errTokenInvalid
	}
	if err != nil {
		return model.UserToken{}, err
	}
	if token.UsedAt != nil || !s.now().Before(token.ExpiresAt) {
		return model.UserToken{}, errTokenInvalid
	}

	used, err := s.userTokens.Use(ctx, token.ID)
	if err != nil {
		return model.UserToken{}, err
	}
	if !used {
		return model.UserToken{}, errTokenInvalid
	}
	return token, nil
}
