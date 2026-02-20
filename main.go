package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Configuration ---
func getPort() string {
	if port := os.Getenv("MOAT_PORT"); port != "" {
		if !strings.HasPrefix(port, ":") {
			return ":" + port
		}
		return port
	}
	if port := os.Getenv("PORT"); port != "" {
		if !strings.HasPrefix(port, ":") {
			return ":" + port
		}
		return port
	}
	return ":8080"
}

// --- Data Models (Simplified ORCID v3 JSON) ---

// TokenResponse represents the OAuth 2.0 response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
	Name         string `json:"name"`
	ORCID        string `json:"orcid"`
}

// OrcidRecord represents the root of the record response
type OrcidRecord struct {
	OrcidIdentifier OrcidIdentifier `json:"orcid-identifier"`
	Person          Person          `json:"person"`
	Activities      Activities      `json:"activities-summary"`
}

type OrcidIdentifier struct {
	Uri  string `json:"uri"`
	Path string `json:"path"`
	Host string `json:"host"`
}

type Person struct {
	Name           Name           `json:"name"`
	Biography      *Biography     `json:"biography,omitempty"`
	Emails         *Emails        `json:"emails,omitempty"`
	ResearcherUrls *ResearcherUrls `json:"researcher-urls,omitempty"`
}

type Biography struct {
	Content string `json:"content"`
}

type Emails struct {
	Email []Email `json:"email"`
}

type Email struct {
	Email string `json:"email"`
	Verified bool `json:"verified"`
	Primary bool `json:"primary"`
	Visibility string `json:"visibility"`
}

type ResearcherUrls struct {
	ResearcherUrl []ResearcherUrl `json:"researcher-url"`
}

type ResearcherUrl struct {
	UrlName string `json:"url-name"`
	Url     Value  `json:"url"`
}

type Name struct {
	GivenNames Value `json:"given-names"`
	FamilyName Value `json:"family-name"`
	CreditName Value `json:"credit-name"`
}

type Value struct {
	Value string `json:"value"`
}

type Activities struct {
	Works      WorkSummaryGroup       `json:"works"`
	Employment EmploymentSummaryGroup `json:"employments"`
}

type WorkSummaryGroup struct {
	Group []WorkGroup `json:"group"`
}

type WorkGroup struct {
	WorkSummary []WorkSummary `json:"work-summary"`
}

type WorkSummary struct {
	PutCode      int          `json:"put-code"`
	Title        Title        `json:"title"`
	Type         string       `json:"type"`
	LastModified LastModified `json:"last-modified-date"`
}

type EmploymentSummaryGroup struct {
	AffiliationGroup []AffiliationGroup `json:"affiliation-group"`
}

type AffiliationGroup struct {
	Summaries []EmploymentSummary `json:"employment-summary"`
}

type EmploymentSummary struct {
	PutCode        int    `json:"put-code"`
	DepartmentName string `json:"department-name"`
	RoleTitle      string `json:"role-title"`
	Organization   Org    `json:"organization"`
}

type Org struct {
	Name string `json:"name"`
}

type Title struct {
	Title Value `json:"title"`
}

type LastModified struct {
	Value int64 `json:"value"` // Unix timestamp
}

// SearchResponse represents a search result
type SearchResponse struct {
	Result   []SearchResult `json:"result"`
	NumFound int            `json:"num-found"`
}

type SearchResult struct {
	OrcidIdentifier OrcidIdentifier `json:"orcid-identifier"`
}

// --- In-Memory Store ---

var (
	// Store works and employments by ORCID -> PutCode -> Data
	// For simplicity, we store raw JSON bytes to mock persistence
	dataStore   = make(map[string]map[string]map[int][]byte)
	personStore = make(map[string]OrcidRecord)
	storeMutex  sync.RWMutex

	// Version is injected at build time
	Version = "dev"
)

func init() {
	// Initialize store with demo users
	people := []struct {
		orcid, given, family, bio string
	}{
		{"0000-0001-2345-6789", "Sofia", "Garcia", "Sofia Garcia is a researcher in the field of Computer Science."},
		{"0000-0002-1001-2002", "John", "Smith", "John Smith studies Physics."},
		{"0000-0003-3003-4004", "Wei", "Chen", "Wei Chen is a Biologist."},
		{"0000-0004-5005-6006", "Priya", "Patel", "Priya Patel works in Chemistry."},
		{"0000-0005-7007-8008", "Ahmed", "Al-Fayed", "Ahmed Al-Fayed is a Mathematician."},
		{"0000-0006-9009-0000", "Elena", "Popov", "Elena Popov researches History."},
	}

	for _, p := range people {
		rec := createMockRecord(p.orcid, p.given, p.family, p.bio)
		personStore[p.orcid] = rec

		dataStore[p.orcid] = map[string]map[int][]byte{
			"work":       make(map[int][]byte),
			"employment": make(map[int][]byte),
		}
	}
}

// --- Handlers ---

func main() {
	// Configure structured logger with Debug level
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
	slog.SetDefault(logger)

	mux := http.NewServeMux()

	// 1. OAuth Token Endpoint
	mux.HandleFunc("POST /oauth/token", handleToken)
	mux.HandleFunc("GET /oauth/authorize", handleAuthorize)

	// 2. Record Retrieval (Public & Member)
	mux.HandleFunc("GET /v3.0/{orcid}/record", handleGetRecord)
	mux.HandleFunc("GET /v3.0/{orcid}/person", handleGetPerson)

	// 3. Works (GET, POST, PUT, DELETE)
	mux.HandleFunc("GET /v3.0/{orcid}/work/{putCode}", handleGetWork)
	mux.HandleFunc("POST /v3.0/{orcid}/work", handlePostWork)
	mux.HandleFunc("PUT /v3.0/{orcid}/work/{putCode}", handlePutWork)

	// 4. Employment (GET, POST, PUT, DELETE)
	mux.HandleFunc("GET /v3.0/{orcid}/employment/{putCode}", handleGetEmployment)
	mux.HandleFunc("POST /v3.0/{orcid}/employment", handlePostEmployment)
	mux.HandleFunc("PUT /v3.0/{orcid}/employment/{putCode}", handlePutEmployment)

	// 5. Search
	mux.HandleFunc("GET /v3.0/search", handleSearch)

	// Middleware for logging and JSON content type
	handler := middleware(mux)

	port := getPort()
	fmt.Printf("ORCID v3 Mock Service running on %s (Version: %s)\n", port, Version)
	fmt.Printf("Try: curl -X POST http://localhost%s/oauth/token -d 'client_id=APP-123&grant_type=client_credentials'\n", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		slog.Error("Unable to start MOAT", "error", err)
	}
}

func middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Try to extract handler name if next is a ServeMux
		handlerName := "unknown"
		if mux, ok := next.(*http.ServeMux); ok {
			if h, pattern := mux.Handler(r); pattern != "" {
				name := runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
				if idx := strings.LastIndex(name, "."); idx != -1 {
					name = name[idx+1:]
				}
				handlerName = name
			}
		}

		// Prepare body for logging
		var bodyLog string
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			if len(bodyBytes) > 0 {
				bodyLog = string(bodyBytes)
			}
		}

		slog.Debug("Handling request",
			"handler-name", handlerName,
			"headers", r.Header,
			"body", bodyLog,
		)

		// Set default headers
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		slog.Info("Request processed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start),
		)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// --- Endpoint Implementations ---

func handleToken(w http.ResponseWriter, r *http.Request) {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Mock response
	resp := TokenResponse{
		AccessToken:  "mock-access-token-12345",
		TokenType:    "bearer",
		RefreshToken: "mock-refresh-token-67890",
		ExpiresIn:    631138518, // ~20 years
		Scope:        "/read-limited /activities/update",
		Name:         "Sofia Garcia",
		ORCID:        "0000-0001-2345-6789",
	}

	json.NewEncoder(w).Encode(resp)
}

func handleAuthorize(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	redirectURI := query.Get("redirect_uri")
	state := query.Get("state")

	if redirectURI == "" {
		http.Error(w, "Missing redirect_uri", http.StatusBadRequest)
		return
	}

	// In a real app, we would show a login screen here.
	// Since this is a mock, we immediately redirect with a code.

	code := "mock-auth-code-12345"
	target := fmt.Sprintf("%s?code=%s", redirectURI, code)
	if state != "" {
		target += "&state=" + state
	}

	http.Redirect(w, r, target, http.StatusFound)
}

func handleGetRecord(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")

	storeMutex.RLock()
	record, ok := personStore[orcid]
	storeMutex.RUnlock()

	if !ok {
		http.Error(w, "Record not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(record)
}

func handleGetPerson(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")

	storeMutex.RLock()
	record, ok := personStore[orcid]
	storeMutex.RUnlock()

	if !ok {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(record.Person)
}

// --- Generic Activity Handlers ---

func handleGetWork(w http.ResponseWriter, r *http.Request) {
	// In a real mock, you'd fetch specific JSON from dataStore
	// Here we return a generic work for any putCode
	putCode, _ := strconv.Atoi(r.PathValue("putCode"))

	response := map[string]interface{}{
		"type":     "work",
		"put-code": putCode,
		"title": map[string]interface{}{
			"title": map[string]string{"value": "Retrieved Mock Work"},
		},
		"publication-date": map[string]interface{}{
			"year": map[string]string{"value": "2023"},
		},
	}

	json.NewEncoder(w).Encode(response)
}

func handlePostWork(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	// Generate a new PutCode
	newPutCode := rand.Intn(999999) + 100000

	// In a real implementation, you would decode the body and save it
	// body, _ := io.ReadAll(r.Body)
	// saveToStore(orcid, "work", newPutCode, body)

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/work/%d", orcid, newPutCode))
	w.WriteHeader(http.StatusCreated)

	// ORCID returns the put-code in the body as well sometimes, or just empty 201
	// We'll mimic returning the put-code for convenience
	json.NewEncoder(w).Encode(map[string]interface{}{
		"put-code": newPutCode,
	})
}

func handlePutWork(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	putCode := r.PathValue("putCode")

	// Update logic would go here

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/work/%s", orcid, putCode))
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"put-code": putCode,
		"status":   "updated",
	})
}

func handleGetEmployment(w http.ResponseWriter, r *http.Request) {
	putCode, _ := strconv.Atoi(r.PathValue("putCode"))

	response := map[string]interface{}{
		"put-code":        putCode,
		"department-name": "Mock Department",
		"role-title":      "Mock Researcher",
		"organization":    map[string]string{"name": "Mock Org"},
		"start-date": map[string]interface{}{
			"year": map[string]string{"value": "2020"},
		},
	}
	json.NewEncoder(w).Encode(response)
}

func handlePostEmployment(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	newPutCode := rand.Intn(999999) + 100000

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/employment/%d", orcid, newPutCode))
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"put-code": newPutCode,
	})
}

func handlePutEmployment(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	putCode := r.PathValue("putCode")

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/employment/%s", orcid, putCode))
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"put-code": putCode,
		"status":   "updated",
	})
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")

	// Simple mock: if query contains "error", return error, else return fake results
	if strings.Contains(query, "error") {
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	resp := SearchResponse{
		NumFound: 1,
		Result: []SearchResult{
			{
				OrcidIdentifier: OrcidIdentifier{
					Uri:  "https://orcid.org/0000-0001-2345-6789",
					Path: "0000-0001-2345-6789",
					Host: "orcid.org",
				},
			},
		},
	}

	json.NewEncoder(w).Encode(resp)
}

func createMockRecord(orcid, givenName, familyName, bio string) OrcidRecord {
	return OrcidRecord{
		OrcidIdentifier: OrcidIdentifier{
			Uri:  fmt.Sprintf("https://orcid.org/%s", orcid),
			Path: orcid,
			Host: "orcid.org",
		},
		Person: Person{
			Name: Name{
				GivenNames: Value{Value: givenName},
				FamilyName: Value{Value: familyName},
				CreditName: Value{Value: fmt.Sprintf("%s. %s", string(givenName[0]), familyName)},
			},
			Biography: &Biography{
				Content: bio,
			},
			Emails: &Emails{
				Email: []Email{
					{
						Email:      fmt.Sprintf("%s.%s@mock.edu", strings.ToLower(givenName), strings.ToLower(familyName)),
						Verified:   true,
						Primary:    true,
						Visibility: "PUBLIC",
					},
				},
			},
			ResearcherUrls: &ResearcherUrls{
				ResearcherUrl: []ResearcherUrl{
					{
						UrlName: "Personal Website",
						Url:     Value{Value: fmt.Sprintf("https://%s.%s.mock", strings.ToLower(givenName), strings.ToLower(familyName))},
					},
				},
			},
		},
		Activities: Activities{
			Works: WorkSummaryGroup{
				Group: []WorkGroup{
					{
						WorkSummary: []WorkSummary{
							{
								PutCode:      123456,
								Title:        Title{Title: Value{Value: "Mock Paper Title"}},
								Type:         "journal-article",
								LastModified: LastModified{Value: time.Now().UnixMilli()},
							},
						},
					},
				},
			},
			Employment: EmploymentSummaryGroup{
				AffiliationGroup: []AffiliationGroup{
					{
						Summaries: []EmploymentSummary{
							{
								PutCode:        789012,
								DepartmentName: "Mock Department",
								RoleTitle:      "Mock Researcher",
								Organization:   Org{Name: "Mock University"},
							},
						},
					},
				},
			},
		},
	}
}
