package handler_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	sqldb "github.com/RamillIslamov/go-from-scratch-project-278/internal/db"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/handler"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/repository"
	"github.com/RamillIslamov/go-from-scratch-project-278/internal/service"
)

type linkResponse struct {
	ID          int64  `json:"id"`
	OriginalURL string `json:"original_url"`
	ShortName   string `json:"short_name"`
	ShortURL    string `json:"short_url"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var (
	testDB     *sql.DB
	testRouter *gin.Engine
)

func TestMain(m *testing.M) {
	_ = godotenv.Load("../../.env.test")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL is empty")
		os.Exit(1)
	}

	dbConn, err := repository.OpenDB(databaseURL)
	if err != nil {
		panic(err)
	}
	testDB = dbConn

	gin.SetMode(gin.TestMode)

	queries := sqldb.New(testDB)
	linksService := service.NewLinksService(queries, "http://localhost:8080")
	linksHandler := handler.NewLinksHandler(linksService)

	router := gin.New()
	router.GET("/r/:code", linksHandler.Redirect)
	api := router.Group("/api")
	{
		api.GET("/links", linksHandler.ListLinks)
		api.POST("/links", linksHandler.CreateLink)
		api.GET("/links/:id", linksHandler.GetLink)
		api.PUT("/links/:id", linksHandler.UpdateLink)
		api.DELETE("/links/:id", linksHandler.DeleteLink)
		api.GET("/link_visits", linksHandler.ListLinkVisits)
	}

	testRouter = router

	code := m.Run()

	_ = testDB.Close()
	os.Exit(code)
}

func resetDB(t *testing.T) {
	t.Helper()

	if _, err := testDB.Exec(`TRUNCATE TABLE link_visits, links RESTART IDENTITY`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
}

func doRequest(t *testing.T, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	return w
}

func TestListLinks_Empty(t *testing.T) {
	resetDB(t)

	w := doRequest(t, http.MethodGet, "/api/links", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	var got []linkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(got) != 0 {
		t.Fatalf("expected empty list, got %+v", got)
	}
}

func TestCreateLink(t *testing.T) {
	resetDB(t)

	body := []byte(`{"original_url":"https://example.com/long-url","short_name":"exmpl"}`)
	w := doRequest(t, http.MethodPost, "/api/links", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, w.Code, w.Body.String())
	}

	var got linkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.ID == 0 {
		t.Fatal("expected non-zero id")
	}
	if got.OriginalURL != "https://example.com/long-url" {
		t.Fatalf("unexpected original_url: %s", got.OriginalURL)
	}
	if got.ShortName != "exmpl" {
		t.Fatalf("unexpected short_name: %s", got.ShortName)
	}
	if got.ShortURL != "http://localhost:8080/r/exmpl" {
		t.Fatalf("unexpected short_url: %s", got.ShortURL)
	}
}

func TestCreateLink_AutoGenerateShortName(t *testing.T) {
	resetDB(t)

	body := []byte(`{"original_url":"https://generated.com"}`)
	w := doRequest(t, http.MethodPost, "/api/links", body)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusCreated, w.Code, w.Body.String())
	}

	var got linkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.ID == 0 {
		t.Fatal("expected non-zero id")
	}
	if got.OriginalURL != "https://generated.com" {
		t.Fatalf("unexpected original_url: %s", got.OriginalURL)
	}
	if got.ShortName == "" {
		t.Fatal("expected generated short_name")
	}
	if got.ShortURL == "" {
		t.Fatal("expected generated short_url")
	}
}

func TestCreateLink_Conflict(t *testing.T) {
	resetDB(t)

	first := []byte(`{"original_url":"https://site1.com","short_name":"same1"}`)
	w1 := doRequest(t, http.MethodPost, "/api/links", first)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected first create status %d, got %d, body=%s", http.StatusCreated, w1.Code, w1.Body.String())
	}

	second := []byte(`{"original_url":"https://site2.com","short_name":"same1"}`)
	w2 := doRequest(t, http.MethodPost, "/api/links", second)

	if w2.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusConflict, w2.Code, w2.Body.String())
	}

	var got errorResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}

	if got.Error != "short_name already exists" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
}

func TestGetLink(t *testing.T) {
	resetDB(t)

	createBody := []byte(`{"original_url":"https://example.com/get","short_name":"get1"}`)
	createW := doRequest(t, http.MethodPost, "/api/links", createBody)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create failed: status=%d body=%s", createW.Code, createW.Body.String())
	}

	var created linkResponse
	if err := json.Unmarshal(createW.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal create response: %v", err)
	}

	getW := doRequest(t, http.MethodGet, "/api/links/1", nil)

	if getW.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, getW.Code, getW.Body.String())
	}

	var got linkResponse
	if err := json.Unmarshal(getW.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal get response: %v", err)
	}

	if got.ID != created.ID {
		t.Fatalf("expected id %d, got %d", created.ID, got.ID)
	}
	if got.OriginalURL != "https://example.com/get" {
		t.Fatalf("unexpected original_url: %s", got.OriginalURL)
	}
	if got.ShortName != "get1" {
		t.Fatalf("unexpected short_name: %s", got.ShortName)
	}
}

func TestGetLink_NotFound(t *testing.T) {
	resetDB(t)

	w := doRequest(t, http.MethodGet, "/api/links/999999", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}

	var got errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.Error != "link not found" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
}

func TestUpdateLink(t *testing.T) {
	resetDB(t)

	createBody := []byte(`{"original_url":"https://example.com/old","short_name":"old1"}`)
	createW := doRequest(t, http.MethodPost, "/api/links", createBody)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create failed: status=%d body=%s", createW.Code, createW.Body.String())
	}

	updateBody := []byte(`{"original_url":"https://example.com/new","short_name":"new1"}`)
	updateW := doRequest(t, http.MethodPut, "/api/links/1", updateBody)

	if updateW.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, updateW.Code, updateW.Body.String())
	}

	var got linkResponse
	if err := json.Unmarshal(updateW.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal update response: %v", err)
	}

	if got.ID != 1 {
		t.Fatalf("expected id 1, got %d", got.ID)
	}
	if got.OriginalURL != "https://example.com/new" {
		t.Fatalf("unexpected original_url: %s", got.OriginalURL)
	}
	if got.ShortName != "new1" {
		t.Fatalf("unexpected short_name: %s", got.ShortName)
	}
	if got.ShortURL != "http://localhost:8080/r/new1" {
		t.Fatalf("unexpected short_url: %s", got.ShortURL)
	}
}

func TestUpdateLink_NotFound(t *testing.T) {
	resetDB(t)

	updateBody := []byte(`{"original_url":"https://example.com/new","short_name":"new1"}`)
	w := doRequest(t, http.MethodPut, "/api/links/999999", updateBody)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}

	var got errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.Error != "link not found" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
}

func TestDeleteLink(t *testing.T) {
	resetDB(t)

	createBody := []byte(`{"original_url":"https://example.com/delete","short_name":"del1"}`)
	createW := doRequest(t, http.MethodPost, "/api/links", createBody)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create failed: status=%d body=%s", createW.Code, createW.Body.String())
	}

	deleteW := doRequest(t, http.MethodDelete, "/api/links/1", nil)

	if deleteW.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusNoContent, deleteW.Code, deleteW.Body.String())
	}

	getW := doRequest(t, http.MethodGet, "/api/links/1", nil)
	if getW.Code != http.StatusNotFound {
		t.Fatalf("expected status %d after delete, got %d, body=%s", http.StatusNotFound, getW.Code, getW.Body.String())
	}
}

func TestDeleteLink_NotFound(t *testing.T) {
	resetDB(t)

	w := doRequest(t, http.MethodDelete, "/api/links/999999", nil)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusNotFound, w.Code, w.Body.String())
	}

	var got errorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if got.Error != "link not found" {
		t.Fatalf("unexpected error: %s", got.Error)
	}
}

func TestListLinks_WithRange(t *testing.T) {
	resetDB(t)

	for i := 0; i < 15; i++ {
		body := []byte(fmt.Sprintf(`{"original_url":"https://example.com/%d","short_name":"s%d"}`, i, i))
		w := doRequest(t, http.MethodPost, "/api/links", body)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: status=%d body=%s", w.Code, w.Body.String())
		}
	}

	w := doRequest(t, http.MethodGet, "/api/links?range=[0,9]", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	contentRange := w.Header().Get("Content-Range")
	if contentRange != "links 0-9/15" {
		t.Fatalf("unexpected Content-Range: %s", contentRange)
	}

	var got []linkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(got) != 10 {
		t.Fatalf("expected 10 items, got %d", len(got))
	}

	if got[0].ID != 1 {
		t.Fatalf("expected first id 1, got %d", got[0].ID)
	}
	if got[9].ID != 10 {
		t.Fatalf("expected last id 10, got %d", got[9].ID)
	}
}

func TestListLinks_WithOffsetRange(t *testing.T) {
	resetDB(t)

	for i := 0; i < 11; i++ {
		body := []byte(fmt.Sprintf(`{"original_url":"https://example.com/%d","short_name":"s%d"}`, i, i))
		w := doRequest(t, http.MethodPost, "/api/links", body)
		if w.Code != http.StatusCreated {
			t.Fatalf("create failed: status=%d body=%s", w.Code, w.Body.String())
		}
	}

	w := doRequest(t, http.MethodGet, "/api/links?range=[5,9]", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, w.Code, w.Body.String())
	}

	contentRange := w.Header().Get("Content-Range")
	if contentRange != "links 5-9/11" {
		t.Fatalf("unexpected Content-Range: %s", contentRange)
	}

	var got []linkResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if len(got) != 5 {
		t.Fatalf("expected 5 items, got %d", len(got))
	}

	if got[0].ID != 6 {
		t.Fatalf("expected first id 6, got %d", got[0].ID)
	}
	if got[4].ID != 10 {
		t.Fatalf("expected last id 10, got %d", got[4].ID)
	}
}

func TestRedirect(t *testing.T) {
	resetDB(t)

	createBody := []byte(`{"original_url":"https://google.com","short_name":"g"}`)
	createW := doRequest(t, http.MethodPost, "/api/links", createBody)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create failed: status=%d body=%s", createW.Code, createW.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/r/g", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "http://localhost:5173")
	req.RemoteAddr = "127.0.0.1:12345"

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusFound, w.Code, w.Body.String())
	}

	location := w.Header().Get("Location")
	if location != "https://google.com" {
		t.Fatalf("unexpected redirect location: %s", location)
	}
}

func TestListLinkVisits(t *testing.T) {
	resetDB(t)

	createBody := []byte(`{"original_url":"https://google.com","short_name":"g"}`)
	createW := doRequest(t, http.MethodPost, "/api/links", createBody)
	if createW.Code != http.StatusCreated {
		t.Fatalf("create failed: status=%d body=%s", createW.Code, createW.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/r/g", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Referer", "http://localhost:5173")
	req.RemoteAddr = "127.0.0.1:12345"

	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("redirect failed: status=%d body=%s", w.Code, w.Body.String())
	}

	listW := doRequest(t, http.MethodGet, "/api/link_visits?range=[0,9]", nil)

	if listW.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d, body=%s", http.StatusOK, listW.Code, listW.Body.String())
	}

	contentRange := listW.Header().Get("Content-Range")
	if contentRange != "link_visits 0-9/1" {
		t.Fatalf("unexpected Content-Range: %s", contentRange)
	}

	var visits []map[string]any
	if err := json.Unmarshal(listW.Body.Bytes(), &visits); err != nil {
		t.Fatalf("unmarshal visits response: %v", err)
	}

	if len(visits) != 1 {
		t.Fatalf("expected 1 visit, got %d", len(visits))
	}
}
