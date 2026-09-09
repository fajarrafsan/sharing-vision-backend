package validation

import (
	"sharing-vision-backend/internal/dto"
	"sharing-vision-backend/internal/model"
)

func ValidateArticle(r dto.ArticleRequest) map[string]string {
	problems := make(map[string]string)

	switch {
	case r.Title == "":
		problems["title"] = "title wajib diisi"
	case len([]rune(r.Title)) < 20:
		problems["title"] = "title minimal 20 karakter"
	case len([]rune(r.Title)) > 200:
		problems["title"] = "title maksimal 200 karakter"
	}

	switch {
	case r.Content == "":
		problems["content"] = "content wajib diisi"
	case len([]rune(r.Content)) < 200:
		problems["content"] = "content minimal 200 karakter"
	}

	switch {
	case r.Category == "":
		problems["category"] = "category wajib diisi"
	case len([]rune(r.Category)) < 3:
		problems["category"] = "category minimal 3 karakter"
	case len([]rune(r.Category)) > 100:
		problems["category"] = "category maksimal 100 karakter"
	}

	switch {
	case r.Status == "":
		problems["status"] = "status wajib diisi"
	case r.Status != model.StatusPublish && r.Status != model.StatusDraft && r.Status != model.StatusThrash:
		problems["status"] = "status harus publish, draft, atau thrash"
	}

	return problems
}
