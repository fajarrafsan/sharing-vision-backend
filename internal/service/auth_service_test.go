package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/model"
	"warta/internal/pagination"
	"warta/internal/repository"
)

// Repository palsu di memori, cukup untuk menguji logika token tanpa MySQL.

type fakeUsers struct {
	byID map[int64]model.User
}

func (f *fakeUsers) Create(_ context.Context, u *model.User) error {
	for _, existing := range f.byID {
		if existing.Email == u.Email {
			return repository.ErrDuplicate
		}
	}
	u.ID = int64(len(f.byID) + 1)
	f.byID[u.ID] = *u
	return nil
}

func (f *fakeUsers) FindByID(_ context.Context, id int64) (model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return model.User{}, repository.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (model.User, error) {
	for _, u := range f.byID {
		if u.Email == email {
			return u, nil
		}
	}
	return model.User{}, repository.ErrNotFound
}

func (f *fakeUsers) List(context.Context, repository.UserFilter, pagination.Params) ([]model.User, int64, error) {
	return nil, 0, nil
}

func (f *fakeUsers) UpdateName(_ context.Context, id int64, name string) error {
	u := f.byID[id]
	u.Name = name
	f.byID[id] = u
	return nil
}

func (f *fakeUsers) UpdatePassword(_ context.Context, id int64, hash string) error {
	u := f.byID[id]
	u.PasswordHash = hash
	f.byID[id] = u
	return nil
}

func (f *fakeUsers) UpdateRole(_ context.Context, id int64, role model.Role) error {
	u := f.byID[id]
	u.Role = role
	f.byID[id] = u
	return nil
}

type fakeTokens struct {
	rows []model.RefreshToken
}

func (f *fakeTokens) Create(_ context.Context, t *model.RefreshToken) error {
	t.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, *t)
	return nil
}

func (f *fakeTokens) FindByHash(_ context.Context, hash string) (model.RefreshToken, error) {
	for _, t := range f.rows {
		if t.TokenHash == hash {
			return t, nil
		}
	}
	return model.RefreshToken{}, repository.ErrNotFound
}

func (f *fakeTokens) Revoke(_ context.Context, id int64) (bool, error) {
	t := &f.rows[id-1]
	if t.RevokedAt != nil {
		return false, nil
	}
	now := time.Now()
	t.RevokedAt = &now
	return true, nil
}

func (f *fakeTokens) RevokeAllForUser(_ context.Context, userID int64) error {
	for i := range f.rows {
		if f.rows[i].UserID == userID && f.rows[i].RevokedAt == nil {
			now := time.Now()
			f.rows[i].RevokedAt = &now
		}
	}
	return nil
}

func (f *fakeTokens) DeleteExpired(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func newAuthService() (*authService, *fakeTokens) {
	tokens := &fakeTokens{}
	s := NewAuthService(
		&fakeUsers{byID: map[int64]model.User{}},
		tokens,
		auth.BcryptHasher{Cost: bcrypt.MinCost},
		auth.NewTokenManager("rahasia-test-yang-panjangnya-lebih-dari-32", "warta", time.Minute),
		time.Hour,
	).(*authService)
	return s, tokens
}

func status(err error) int {
	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return 0
}

func TestRefreshRotation(t *testing.T) {
	s, _ := newAuthService()
	ctx := context.Background()

	first, err := s.Register(ctx, dto.RegisterRequest{Name: "Budi", Email: "budi@warta.test", Password: "rahasia123"})
	if err != nil {
		t.Fatal(err)
	}

	second, err := s.Refresh(ctx, dto.RefreshRequest{RefreshToken: first.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Fatal("refresh token seharusnya dirotasi")
	}

	// Token lama dipakai lagi: ditolak, dan token terbaru ikut dicabut.
	if _, err := s.Refresh(ctx, dto.RefreshRequest{RefreshToken: first.RefreshToken}); status(err) != 401 {
		t.Fatalf("token lama: %v", err)
	}
	if _, err := s.Refresh(ctx, dto.RefreshRequest{RefreshToken: second.RefreshToken}); status(err) != 401 {
		t.Fatalf("token terbaru seharusnya ikut dicabut: %v", err)
	}
}

func TestRefreshExpired(t *testing.T) {
	s, _ := newAuthService()
	ctx := context.Background()

	pair, err := s.Register(ctx, dto.RegisterRequest{Name: "Budi", Email: "budi@warta.test", Password: "rahasia123"})
	if err != nil {
		t.Fatal(err)
	}

	s.now = func() time.Time { return time.Now().Add(2 * time.Hour) }
	if _, err := s.Refresh(ctx, dto.RefreshRequest{RefreshToken: pair.RefreshToken}); status(err) != 401 {
		t.Fatalf("token kedaluwarsa: %v", err)
	}
}

func TestLoginAndPasswordChange(t *testing.T) {
	s, tokens := newAuthService()
	ctx := context.Background()

	pair, err := s.Register(ctx, dto.RegisterRequest{Name: "Budi", Email: "Budi@Warta.test", Password: "rahasia123"})
	if err != nil {
		t.Fatal(err)
	}
	if pair.User.Role != string(model.RoleReader) {
		t.Fatalf("role awal: %s", pair.User.Role)
	}

	if _, err := s.Login(ctx, dto.LoginRequest{Email: "budi@warta.test", Password: "salah-sekali"}); status(err) != 401 {
		t.Fatalf("password salah: %v", err)
	}
	if _, err := s.Login(ctx, dto.LoginRequest{Email: " BUDI@warta.test ", Password: "rahasia123"}); err != nil {
		t.Fatalf("email tidak peka huruf besar: %v", err)
	}

	actor := auth.Actor{ID: pair.User.ID, Role: model.RoleReader}
	err = s.ChangePassword(ctx, actor, dto.ChangePasswordRequest{CurrentPassword: "rahasia123", NewPassword: "rahasia456"})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range tokens.rows {
		if row.RevokedAt == nil {
			t.Fatal("semua refresh token seharusnya dicabut setelah ganti password")
		}
	}
	if _, err := s.Login(ctx, dto.LoginRequest{Email: "budi@warta.test", Password: "rahasia456"}); err != nil {
		t.Fatalf("login dengan password baru: %v", err)
	}
}

func TestEnsureAdmin(t *testing.T) {
	s, _ := newAuthService()
	ctx := context.Background()

	pair, _ := s.Register(ctx, dto.RegisterRequest{Name: "Budi", Email: "budi@warta.test", Password: "rahasia123"})
	if err := s.EnsureAdmin(ctx, "Budi", "budi@warta.test", "apa-saja-123"); err != nil {
		t.Fatal(err)
	}

	me, _ := s.Me(ctx, auth.Actor{ID: pair.User.ID})
	if me.Role != string(model.RoleAdmin) {
		t.Fatalf("akun yang sudah ada seharusnya dinaikkan menjadi admin: %s", me.Role)
	}
	// Password akun yang sudah ada tidak diubah.
	if _, err := s.Login(ctx, dto.LoginRequest{Email: "budi@warta.test", Password: "rahasia123"}); err != nil {
		t.Fatalf("password lama: %v", err)
	}
}
