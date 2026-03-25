package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/db"
	"math/rand"
	"strings"
	"time"

	"github.com/lib/pq"
)

var ErrNotFound = errors.New("link not found")
var ErrShortNameConflict = errors.New("short name already exists")

type Link struct {
	ID          int64     `json:"id"`
	OriginalURL string    `json:"original_url"`
	ShortName   string    `json:"short_name"`
	ShortURL    string    `json:"short_url"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateLinkInput struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

type UpdateLinkInput struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

type LinksService struct {
	queries    *db.Queries
	appBaseURL string
}

func NewLinksService(queries *db.Queries, appBaseURL string) *LinksService {
	return &LinksService{
		queries:    queries,
		appBaseURL: strings.TrimRight(appBaseURL, "/"),
	}
}

func (s *LinksService) List() ([]Link, error) {
	rows, err := s.queries.ListLinks(context.Background())
	if err != nil {
		return nil, err
	}

	result := make([]Link, 0, len(rows))
	for _, row := range rows {
		result = append(result, Link{
			ID:          row.ID,
			OriginalURL: row.OriginalUrl,
			ShortName:   row.ShortName,
			ShortURL:    fmt.Sprintf("%s/r/%s", s.appBaseURL, row.ShortName),
			CreatedAt:   row.CreatedAt,
		})
	}

	return result, nil
}

func (s *LinksService) Get(id int64) (Link, error) {
	row, err := s.queries.GetLink(context.Background(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Link{}, ErrNotFound
		}
		return Link{}, err
	}

	return Link{
		ID:          row.ID,
		OriginalURL: row.OriginalUrl,
		ShortName:   row.ShortName,
		ShortURL:    fmt.Sprintf("%s/r/%s", s.appBaseURL, row.ShortName),
		CreatedAt:   row.CreatedAt,
	}, nil
}

func (s *LinksService) Create(input CreateLinkInput) (Link, error) {
	shortName := strings.TrimSpace(input.ShortName)
	if shortName == "" {
		shortName = generateShortName(6)
	}

	row, err := s.queries.CreateLink(context.Background(), db.CreateLinkParams{
		OriginalUrl: input.OriginalURL,
		ShortName:   shortName,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return Link{}, ErrShortNameConflict
		}
		return Link{}, err
	}

	return Link{
		ID:          row.ID,
		OriginalURL: row.OriginalUrl,
		ShortName:   row.ShortName,
		ShortURL:    fmt.Sprintf("%s/r/%s", s.appBaseURL, row.ShortName),
		CreatedAt:   row.CreatedAt,
	}, nil
}

func (s *LinksService) Update(id int64, input UpdateLinkInput) (Link, error) {
	row, err := s.queries.UpdateLink(context.Background(), db.UpdateLinkParams{
		ID:          id,
		OriginalUrl: input.OriginalURL,
		ShortName:   input.ShortName,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Link{}, ErrNotFound
		}
		if isUniqueViolation(err) {
			return Link{}, ErrShortNameConflict
		}
		return Link{}, err
	}

	return Link{
		ID:          row.ID,
		OriginalURL: row.OriginalUrl,
		ShortName:   row.ShortName,
		ShortURL:    fmt.Sprintf("%s/r/%s", s.appBaseURL, row.ShortName),
		CreatedAt:   row.CreatedAt,
	}, nil
}

func (s *LinksService) Delete(id int64) error {
	affected, err := s.queries.DeleteLink(context.Background(), id)
	if err != nil {
		return err
	}

	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateShortName(n int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}
