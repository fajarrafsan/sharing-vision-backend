package service

import (
	"context"
	"errors"
	"time"

	"warta/internal/apperr"
	"warta/internal/auth"
	"warta/internal/dto"
	"warta/internal/model"
	"warta/internal/pagination"
	"warta/internal/repository"
	"warta/internal/validation"
)

type CommentService interface {
	List(ctx context.Context, actor auth.Actor, articleID int64, p pagination.Params) ([]dto.CommentResponse, pagination.Meta, error)
	Create(ctx context.Context, actor auth.Actor, articleID int64, req dto.CommentRequest) (dto.CommentResponse, error)
	Update(ctx context.Context, actor auth.Actor, id int64, req dto.CommentRequest) (dto.CommentResponse, error)
	Delete(ctx context.Context, actor auth.Actor, id int64) error

	// Report mencatat laporan pembaca. Komentar disembunyikan otomatis bila
	// laporannya mencapai ambang, sampai admin meninjaunya.
	Report(ctx context.Context, actor auth.Actor, id int64, req dto.ReportRequest) (dto.ReportResponse, error)
	// ListReported adalah antrean moderasi untuk admin.
	ListReported(ctx context.Context, p pagination.Params) ([]dto.ReportedCommentResponse, pagination.Meta, error)
	// Moderate: approve menampilkan kembali dan menghapus laporannya, hide
	// menyembunyikan komentar.
	Moderate(ctx context.Context, id int64, req dto.ModerationRequest) error
}

// duplicateWindow adalah rentang waktu komentar yang sama persis dari akun
// yang sama dianggap kiriman ganda.
const duplicateWindow = 10 * time.Minute

type commentService struct {
	comments      repository.CommentRepository
	articles      repository.ArticleRepository
	hideThreshold int
	now           func() time.Time
}

func NewCommentService(comments repository.CommentRepository, articles repository.ArticleRepository, hideThreshold int) CommentService {
	return &commentService{comments: comments, articles: articles, hideThreshold: hideThreshold, now: time.Now}
}

var errCommentNotFound = apperr.NotFound("komentar tidak ditemukan")

func (s *commentService) List(ctx context.Context, actor auth.Actor, articleID int64, p pagination.Params) ([]dto.CommentResponse, pagination.Meta, error) {
	if _, err := s.visibleArticle(ctx, actor, articleID); err != nil {
		return nil, pagination.Meta{}, err
	}

	comments, total, err := s.comments.ListByArticle(ctx, articleID, p)
	if err != nil {
		return nil, pagination.Meta{}, apperr.Internal(err)
	}
	return dto.NewCommentResponses(comments), pagination.NewMeta(p, total), nil
}

func (s *commentService) Create(ctx context.Context, actor auth.Actor, articleID int64, req dto.CommentRequest) (dto.CommentResponse, error) {
	article, err := s.visibleArticle(ctx, actor, articleID)
	if err != nil {
		return dto.CommentResponse{}, err
	}
	if !article.IsPublished() {
		return dto.CommentResponse{}, apperr.Forbidden("komentar hanya bisa ditambahkan ke article yang sudah terbit")
	}

	req.Normalize()
	if problems := validation.ValidateComment(req); len(problems) > 0 {
		return dto.CommentResponse{}, apperr.Validation(problems)
	}

	duplicate, err := s.comments.HasRecentDuplicate(ctx, articleID, actor.ID, req.Body, s.now().Add(-duplicateWindow))
	if err != nil {
		return dto.CommentResponse{}, apperr.Internal(err)
	}
	if duplicate {
		return dto.CommentResponse{}, apperr.Conflict("komentar yang sama baru saja kamu kirim")
	}

	comment := model.Comment{ArticleID: articleID, UserID: actor.ID, Body: req.Body}
	if err := s.comments.Create(ctx, &comment); err != nil {
		if errors.Is(err, repository.ErrMissingReference) {
			return dto.CommentResponse{}, errArticleNotFound
		}
		return dto.CommentResponse{}, apperr.Internal(err)
	}

	return dto.NewCommentResponse(comment), nil
}

// Update hanya untuk pemilik komentar. Admin boleh menghapus komentar orang
// lain, tapi tidak mengubah kata-katanya.
func (s *commentService) Update(ctx context.Context, actor auth.Actor, id int64, req dto.CommentRequest) (dto.CommentResponse, error) {
	comment, err := s.find(ctx, id)
	if err != nil {
		return dto.CommentResponse{}, err
	}
	if comment.UserID != actor.ID {
		return dto.CommentResponse{}, apperr.Forbidden("hanya penulis komentar yang bisa mengubahnya")
	}

	req.Normalize()
	if problems := validation.ValidateComment(req); len(problems) > 0 {
		return dto.CommentResponse{}, apperr.Validation(problems)
	}

	if err := s.comments.UpdateBody(ctx, id, req.Body); err != nil {
		return dto.CommentResponse{}, apperr.Internal(err)
	}

	updated, err := s.find(ctx, id)
	if err != nil {
		return dto.CommentResponse{}, err
	}
	return dto.NewCommentResponse(updated), nil
}

func (s *commentService) Delete(ctx context.Context, actor auth.Actor, id int64) error {
	comment, err := s.find(ctx, id)
	if err != nil {
		return err
	}
	if comment.UserID != actor.ID && !actor.IsAdmin() {
		return apperr.Forbidden("hanya penulis komentar dan admin yang bisa menghapusnya")
	}

	err = s.comments.Delete(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return errCommentNotFound
	}
	if err != nil {
		return apperr.Internal(err)
	}
	return nil
}

func (s *commentService) find(ctx context.Context, id int64) (model.Comment, error) {
	comment, err := s.comments.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Comment{}, errCommentNotFound
	}
	if err != nil {
		return model.Comment{}, apperr.Internal(err)
	}
	return comment, nil
}

func (s *commentService) visibleArticle(ctx context.Context, actor auth.Actor, id int64) (model.Article, error) {
	article, err := s.articles.FindByID(ctx, id)
	if errors.Is(err, repository.ErrNotFound) || err == nil && !canView(actor, article) {
		return model.Article{}, errArticleNotFound
	}
	if err != nil {
		return model.Article{}, apperr.Internal(err)
	}
	return article, nil
}

func (s *commentService) Report(ctx context.Context, actor auth.Actor, id int64, req dto.ReportRequest) (dto.ReportResponse, error) {
	req.Normalize()
	if problems := validation.ValidateReport(req); len(problems) > 0 {
		return dto.ReportResponse{}, apperr.Validation(problems)
	}

	comment, err := s.find(ctx, id)
	if err != nil {
		return dto.ReportResponse{}, err
	}
	if _, err := s.visibleArticle(ctx, actor, comment.ArticleID); err != nil {
		return dto.ReportResponse{}, errCommentNotFound
	}
	if comment.HiddenAt != nil {
		// Sudah disembunyikan dan menunggu admin; laporan tambahan tidak perlu.
		return dto.ReportResponse{Reported: true, Hidden: true}, nil
	}
	if comment.UserID == actor.ID {
		return dto.ReportResponse{}, apperr.Forbidden("tidak bisa melaporkan komentar sendiri")
	}

	count, err := s.comments.Report(ctx, id, actor.ID, req.Reason)
	if errors.Is(err, repository.ErrMissingReference) {
		return dto.ReportResponse{}, errCommentNotFound
	}
	if err != nil {
		return dto.ReportResponse{}, apperr.Internal(err)
	}

	hidden := count >= s.hideThreshold
	if hidden {
		if err := s.comments.SetHidden(ctx, id, true); err != nil {
			return dto.ReportResponse{}, apperr.Internal(err)
		}
	}
	return dto.ReportResponse{Reported: true, Hidden: hidden}, nil
}

func (s *commentService) ListReported(ctx context.Context, p pagination.Params) ([]dto.ReportedCommentResponse, pagination.Meta, error) {
	comments, total, err := s.comments.ListReported(ctx, p)
	if err != nil {
		return nil, pagination.Meta{}, apperr.Internal(err)
	}
	return dto.NewReportedCommentResponses(comments), pagination.NewMeta(p, total), nil
}

func (s *commentService) Moderate(ctx context.Context, id int64, req dto.ModerationRequest) error {
	req.Normalize()
	if problems := validation.ValidateModeration(req); len(problems) > 0 {
		return apperr.Validation(problems)
	}
	if _, err := s.find(ctx, id); err != nil {
		return err
	}

	if req.Action == "hide" {
		if err := s.comments.SetHidden(ctx, id, true); err != nil {
			return apperr.Internal(err)
		}
		return nil
	}

	if err := s.comments.ClearReports(ctx, id); err != nil {
		return apperr.Internal(err)
	}
	if err := s.comments.SetHidden(ctx, id, false); err != nil {
		return apperr.Internal(err)
	}
	return nil
}
