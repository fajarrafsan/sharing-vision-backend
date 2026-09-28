package validation

import (
	"strings"
	"testing"

	"warta/internal/dto"
)

func TestValidateArticle(t *testing.T) {
	valid := dto.ArticleRequest{
		Title:      "Judul yang cukup panjang untuk lolos",
		Content:    strings.Repeat("a", 200),
		CategoryID: 1,
		Tags:       []string{"go", "backend"},
		Status:     "draft",
	}
	if problems := ValidateArticle(valid); len(problems) != 0 {
		t.Fatalf("request sah ditolak: %v", problems)
	}

	tests := map[string]func(r *dto.ArticleRequest){
		"title":       func(r *dto.ArticleRequest) { r.Title = "Pendek" },
		"content":     func(r *dto.ArticleRequest) { r.Content = strings.Repeat("a", 199) },
		"category_id": func(r *dto.ArticleRequest) { r.CategoryID = 0 },
		"status":      func(r *dto.ArticleRequest) { r.Status = "publish" },
		"tags":        func(r *dto.ArticleRequest) { r.Tags = strings.Fields(strings.Repeat("tag ", 11)) },
	}
	for field, mutate := range tests {
		r := valid
		mutate(&r)
		if problems := ValidateArticle(r); problems[field] == "" || len(problems) != 1 {
			t.Errorf("%s: %v", field, problems)
		}
	}

	// Batas dihitung per karakter, bukan per byte.
	r := valid
	r.Title = strings.Repeat("é", 20)
	if problems := ValidateArticle(r); problems["title"] != "" {
		t.Errorf("20 karakter multibyte: %v", problems)
	}
}

func TestValidateRegister(t *testing.T) {
	valid := dto.RegisterRequest{Name: "Budi", Email: "budi@warta.test", Password: "rahasia123"}
	if problems := ValidateRegister(valid); len(problems) != 0 {
		t.Fatalf("request sah ditolak: %v", problems)
	}

	for _, email := range []string{"budi", "Budi <budi@warta.test>", "budi@", strings.Repeat("a", 190) + "@x.id"} {
		r := valid
		r.Email = email
		if ValidateRegister(r)["email"] == "" {
			t.Errorf("email %q seharusnya ditolak", email)
		}
	}

	r := valid
	r.Password = strings.Repeat("a", 73)
	if ValidateRegister(r)["password"] == "" {
		t.Error("password lebih dari 72 byte seharusnya ditolak")
	}
}

func TestValidateChangePassword(t *testing.T) {
	problems := ValidateChangePassword(dto.ChangePasswordRequest{CurrentPassword: "rahasia123", NewPassword: "rahasia123"})
	if problems["new_password"] == "" {
		t.Fatal("password baru yang sama seharusnya ditolak")
	}
}
