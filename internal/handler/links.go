package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/service"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type LinksHandler struct {
	service *service.LinksService
}

func NewLinksHandler(service *service.LinksService) *LinksHandler {
	return &LinksHandler{service: service}
}

type linkRequest struct {
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
}

func (h *LinksHandler) ListLinks(c *gin.Context) {
	from, to, err := parseRange(c.Query("range"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range"})
		return
	}

	result, err := h.service.List(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list links"})
		return
	}

	c.Header("Accept-Ranges", "links")
	c.Header("Content-Range", fmt.Sprintf("links %d-%d/%d", from, to, result.Total))
	c.JSON(http.StatusOK, result.Links)
}

func (h *LinksHandler) GetLink(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	link, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get link"})
		return
	}

	c.JSON(http.StatusOK, link)
}

func (h *LinksHandler) CreateLink(c *gin.Context) {
	var req linkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.OriginalURL = strings.TrimSpace(req.OriginalURL)
	req.ShortName = strings.TrimSpace(req.ShortName)

	if req.OriginalURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "original_url is required"})
		return
	}

	link, err := h.service.Create(service.CreateLinkInput{
		OriginalURL: req.OriginalURL,
		ShortName:   req.ShortName,
	})
	if err != nil {
		if errors.Is(err, service.ErrShortNameConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "short_name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create link"})
		return
	}

	c.JSON(http.StatusCreated, link)
}

func (h *LinksHandler) UpdateLink(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req linkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	req.OriginalURL = strings.TrimSpace(req.OriginalURL)
	req.ShortName = strings.TrimSpace(req.ShortName)

	if req.OriginalURL == "" || req.ShortName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "original_url and short_name are required"})
		return
	}

	link, err := h.service.Update(id, service.UpdateLinkInput{
		OriginalURL: req.OriginalURL,
		ShortName:   req.ShortName,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		if errors.Is(err, service.ErrShortNameConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "short_name already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update link"})
		return
	}

	c.JSON(http.StatusOK, link)
}

func (h *LinksHandler) DeleteLink(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	err = h.service.Delete(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete link"})
		return
	}

	c.Status(http.StatusNoContent)
}

func parseID(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

func parseRange(raw string) (int32, int32, error) {
	if strings.TrimSpace(raw) == "" {
		return 0, 9, nil
	}

	var values []int32
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return 0, 0, err
	}

	if len(values) != 2 {
		return 0, 0, fmt.Errorf("range must contain 2 values")
	}

	from := values[0]
	to := values[1]

	if from < 0 || to < 0 || to < from {
		return 0, 0, fmt.Errorf("invalid range")
	}

	return from, to, nil
}

func (h *LinksHandler) Redirect(c *gin.Context) {
	code := c.Param("code")

	link, err := h.service.GetByShortName(code)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "link not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find link"})
		return
	}

	status := int32(http.StatusFound)

	_, visitErr := h.service.CreateVisit(
		link.ID,
		c.ClientIP(),
		c.GetHeader("User-Agent"),
		c.GetHeader("Referer"),
		status,
	)
	if visitErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save link visit"})
		return
	}

	c.Redirect(http.StatusFound, link.OriginalURL)
}

func (h *LinksHandler) ListLinkVisits(c *gin.Context) {
	from, to, err := parseRange(c.Query("range"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range"})
		return
	}

	result, err := h.service.ListVisits(int64(from), int64(to))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list link visits"})
		return
	}

	c.Header("Accept-Ranges", "link_visits")
	c.Header("Content-Range", fmt.Sprintf("link_visits %d-%d/%d", from, to, result.Total))
	c.JSON(http.StatusOK, result.Visits)
}
