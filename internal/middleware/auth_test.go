package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(token string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(token))
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	return r
}

func TestAuth_DisabledWhenTokenEmpty(t *testing.T) {
	r := setupRouter("")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 when auth disabled, got %d", w.Code)
	}
}

func TestAuth_RejectsMissingHeader(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no header, got %d", w.Code)
	}
}

func TestAuth_RejectsWrongToken(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong token, got %d", w.Code)
	}
}

func TestAuth_RejectsMalformedHeader(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "secret") // missing Bearer prefix
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with malformed header, got %d", w.Code)
	}
}

func TestAuth_AcceptsValidToken(t *testing.T) {
	r := setupRouter("secret")
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer secret")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid token, got %d", w.Code)
	}
}
