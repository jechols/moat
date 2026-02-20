package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHandleAuthorize(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("GET", "/oauth/authorize?redirect_uri=http://example.com", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Errorf("Expected status Found, got %v", w.Code)
	}

	loc := w.Header().Get("Location")
	if !strings.Contains(loc, "http://example.com") {
		t.Errorf("Expected redirect to example.com, got %s", loc)
	}
	if !strings.Contains(loc, "code=") {
		t.Errorf("Expected code in redirect params, got %s", loc)
	}
}

func TestHandleAuthorizeMissingRedirect(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("GET", "/oauth/authorize", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status Bad Request, got %v", w.Code)
	}
}

func TestHandleToken(t *testing.T) {
	handler := setupRouter()

	data := url.Values{}
	data.Set("client_id", "APP-123")
	data.Set("grant_type", "client_credentials")

	req := httptest.NewRequest("POST", "/oauth/token", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var resp TokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.AccessToken == "" {
		t.Error("Expected access token in response")
	}
}

func TestHandleGetRecord(t *testing.T) {
	handler := setupRouter()
	orcid := "0000-0001-2345-6789" // Sofia Garcia

	req := httptest.NewRequest("GET", "/v3.0/"+orcid+"/record", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var rec OrcidRecord
	if err := json.NewDecoder(w.Body).Decode(&rec); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if rec.OrcidIdentifier.Path != orcid {
		t.Errorf("Expected ORCID %s, got %s", orcid, rec.OrcidIdentifier.Path)
	}
}

func TestHandleGetPerson(t *testing.T) {
	handler := setupRouter()
	orcid := "0000-0001-2345-6789"

	req := httptest.NewRequest("GET", "/v3.0/"+orcid+"/person", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var person Person
	if err := json.NewDecoder(w.Body).Decode(&person); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if person.Name.GivenNames.Value != "Sofia" {
		t.Errorf("Expected given name Sofia, got %s", person.Name.GivenNames.Value)
	}
}

func TestHandleGetWork(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("GET", "/v3.0/0000-0001-2345-6789/work/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestHandlePostWork(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("POST", "/v3.0/0000-0001-2345-6789/work", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}

	if w.Header().Get("Location") == "" {
		t.Error("Expected Location header")
	}
}

func TestHandlePutWork(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("PUT", "/v3.0/0000-0001-2345-6789/work/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestHandleGetEmployment(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("GET", "/v3.0/0000-0001-2345-6789/employment/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestHandlePostEmployment(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("POST", "/v3.0/0000-0001-2345-6789/employment", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status Created, got %v", w.Code)
	}

	if w.Header().Get("Location") == "" {
		t.Error("Expected Location header")
	}
}

func TestHandlePutEmployment(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("PUT", "/v3.0/0000-0001-2345-6789/employment/123", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}
}

func TestHandleSearch(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("GET", "/v3.0/search?q=test", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	var resp SearchResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if resp.NumFound != 1 {
		t.Errorf("Expected 1 result, got %d", resp.NumFound)
	}
}

func TestHandleSearchError(t *testing.T) {
	handler := setupRouter()
	req := httptest.NewRequest("GET", "/v3.0/search?q=error", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %v", w.Code)
	}
}
