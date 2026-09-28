package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"warta/internal/model"
)

const secret = "rahasia-test-yang-panjangnya-lebih-dari-32"

func TestIssueAndParse(t *testing.T) {
	m := NewTokenManager(secret, "warta", time.Minute)

	token, err := m.Issue(model.User{ID: 7, Role: model.RoleAuthor})
	if err != nil {
		t.Fatal(err)
	}

	actor, err := m.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if actor.ID != 7 || actor.Role != model.RoleAuthor || !actor.CanWrite() || actor.IsAdmin() {
		t.Fatalf("actor: %+v", actor)
	}
}

func TestParseRejects(t *testing.T) {
	m := NewTokenManager(secret, "warta", time.Minute)
	valid, _ := m.Issue(model.User{ID: 1, Role: model.RoleReader})

	expired := NewTokenManager(secret, "warta", time.Minute)
	expired.now = func() time.Time { return time.Now().Add(-time.Hour) }
	old, _ := expired.Issue(model.User{ID: 1, Role: model.RoleReader})
	if _, err := m.Parse(old); !errors.Is(err, ErrExpiredToken) {
		t.Errorf("token kedaluwarsa: %v", err)
	}

	otherSecret, _ := NewTokenManager(secret+"x", "warta", time.Minute).Issue(model.User{ID: 1, Role: model.RoleReader})
	otherIssuer, _ := NewTokenManager(secret, "lain", time.Minute).Issue(model.User{ID: 1, Role: model.RoleReader})
	badRole, _ := m.Issue(model.User{ID: 1, Role: "superuser"})

	// alg none tidak boleh diterima walau klaimnya sah.
	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "1", "role": "admin", "iss": "warta", "exp": time.Now().Add(time.Minute).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)

	for name, token := range map[string]string{
		"secret lain": otherSecret,
		"issuer lain": otherIssuer,
		"role asing":  badRole,
		"alg none":    none,
		"rusak":       valid[:len(valid)-2],
		"kosong":      "",
	} {
		if _, err := m.Parse(token); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestRefreshToken(t *testing.T) {
	plain, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(plain) < 40 || HashRefreshToken(plain) != hash || len(hash) != 64 {
		t.Fatalf("plain %q hash %q", plain, hash)
	}

	other, _, _ := NewRefreshToken()
	if other == plain {
		t.Fatal("token acak tidak boleh sama")
	}
}
