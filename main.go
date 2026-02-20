package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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
	XMLName         xml.Name        `json:"-" xml:"record"`
	OrcidIdentifier OrcidIdentifier `json:"orcid-identifier" xml:"orcid-identifier"`
	Person          Person          `json:"person" xml:"person"`
	Activities      Activities      `json:"activities-summary" xml:"activities-summary"`
}

type OrcidIdentifier struct {
	Uri  string `json:"uri" xml:"uri"`
	Path string `json:"path" xml:"path"`
	Host string `json:"host" xml:"host"`
}

type Person struct {
	Name           Name            `json:"name" xml:"name"`
	Biography      *Biography      `json:"biography,omitempty" xml:"biography,omitempty"`
	Emails         *Emails         `json:"emails,omitempty" xml:"emails,omitempty"`
	ResearcherUrls *ResearcherUrls `json:"researcher-urls,omitempty" xml:"researcher-urls,omitempty"`
}

type Biography struct {
	Content string `json:"content" xml:"content"`
}

type Emails struct {
	Email []Email `json:"email" xml:"email"`
}

type Email struct {
	Email      string `json:"email" xml:"email"`
	Verified   bool   `json:"verified" xml:"verified"`
	Primary    bool   `json:"primary" xml:"primary"`
	Visibility string `json:"visibility" xml:"visibility"`
}

type ResearcherUrls struct {
	ResearcherUrl []ResearcherUrl `json:"researcher-url" xml:"researcher-url"`
}

type ResearcherUrl struct {
	UrlName string `json:"url-name" xml:"url-name"`
	Url     Value  `json:"url" xml:"url"`
}

type Name struct {
	GivenNames Value `json:"given-names" xml:"given-names"`
	FamilyName Value `json:"family-name" xml:"family-name"`
	CreditName Value `json:"credit-name" xml:"credit-name"`
}

type Value struct {
	Value string `json:"value" xml:"value"`
}

type Activities struct {
	Works      WorkSummaryGroup       `json:"works" xml:"works"`
	Employment EmploymentSummaryGroup `json:"employments" xml:"employments"`
}

type WorkSummaryGroup struct {
	Group []WorkGroup `json:"group" xml:"group"`
}

type WorkGroup struct {
	WorkSummary []WorkSummary `json:"work-summary" xml:"work-summary"`
}

type WorkSummary struct {
	PutCode      int          `json:"put-code" xml:"put-code"`
	Title        Title        `json:"title" xml:"title"`
	Type         string       `json:"type" xml:"type"`
	LastModified LastModified `json:"last-modified-date" xml:"last-modified-date"`
}

type EmploymentSummaryGroup struct {
	AffiliationGroup []AffiliationGroup `json:"affiliation-group" xml:"affiliation-group"`
}

type AffiliationGroup struct {
	Summaries []EmploymentSummary `json:"employment-summary" xml:"employment-summary"`
}

type EmploymentSummary struct {
	PutCode        int    `json:"put-code" xml:"put-code"`
	DepartmentName string `json:"department-name" xml:"department-name"`
	RoleTitle      string `json:"role-title" xml:"role-title"`
	Organization   Org    `json:"organization" xml:"organization"`
}

type Org struct {
	Name string `json:"name" xml:"name"`
}

type Title struct {
	Title Value `json:"title" xml:"title"`
}

type LastModified struct {
	Value int64 `json:"value" xml:"value"` // Unix timestamp
}

// SearchResponse represents a search result
type SearchResponse struct {
	XMLName  xml.Name       `json:"-" xml:"search:search"`
	Result   []SearchResult `json:"result" xml:"result"`
	NumFound int            `json:"num-found" xml:"num-found"`
}

type SearchResult struct {
	OrcidIdentifier OrcidIdentifier `json:"orcid-identifier" xml:"orcid-identifier"`
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

	// Middleware for logging and content type
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

		// We do NOT set default Content-Type here anymore, because it depends on the endpoint and accept header.
		// However, we can set a safe default like JSON if we want, but writeResponse will override it.
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

// writeResponse handles content negotiation for /v3.0/ endpoints
func writeResponse(w http.ResponseWriter, r *http.Request, data interface{}) {
	accept := r.Header.Get("Accept")

	// Check if request is for /v3.0/
	isV3 := strings.HasPrefix(r.URL.Path, "/v3.0/")

	// Default to JSON unless it's v3.0 AND (JSON is NOT explicitly requested OR XML IS requested)
	// Actually, easier logic:
	// If Accept contains "json", use JSON.
	// Else if isV3, use XML.
	// Else default to JSON.

	useXML := false
	if isV3 {
		if strings.Contains(accept, "application/json") {
			useXML = false
		} else {
			// If JSON is not requested, default to XML for v3.0
			useXML = true
		}
	} else {
		// Non-v3 endpoints (like oauth) default to JSON
		useXML = false
	}

	if useXML {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Write([]byte(xml.Header))
		if err := xml.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Failed to encode XML response", "error", err)
		}
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			slog.Error("Failed to encode JSON response", "error", err)
		}
	}
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

	// Token endpoint always returns JSON
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
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

	writeResponse(w, r, record)
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

	writeResponse(w, r, record.Person)
}

// --- Generic Activity Handlers ---

// Helper struct for generic responses (needs XML tags too)
type GenericWorkResponse struct {
	XMLName         xml.Name `json:"-" xml:"work:work"`
	Type            string   `json:"type" xml:"type"`
	PutCode         int      `json:"put-code" xml:"put-code"`
	Title           Title    `json:"title" xml:"title"`
	PublicationDate DateYear `json:"publication-date" xml:"publication-date"`
}

type DateYear struct {
	Year Value `json:"year" xml:"year"`
}

func handleGetWork(w http.ResponseWriter, r *http.Request) {
	// In a real mock, you'd fetch specific JSON from dataStore
	// Here we return a generic work for any putCode
	putCode, _ := strconv.Atoi(r.PathValue("putCode"))

	response := GenericWorkResponse{
		Type:    "work",
		PutCode: putCode,
		Title: Title{
			Title: Value{Value: "Retrieved Mock Work"},
		},
		PublicationDate: DateYear{
			Year: Value{Value: "2023"},
		},
	}

	writeResponse(w, r, response)
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
	// Create a simple struct for this response
	type PutCodeResponse struct {
		XMLName xml.Name `json:"-" xml:"response"`
		PutCode int      `json:"put-code" xml:"put-code"`
	}

	writeResponse(w, r, PutCodeResponse{PutCode: newPutCode})
}

func handlePutWork(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	putCode := r.PathValue("putCode")

	// Update logic would go here

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/work/%s", orcid, putCode))
	w.WriteHeader(http.StatusOK)

	type UpdateResponse struct {
		XMLName xml.Name `json:"-" xml:"response"`
		PutCode string   `json:"put-code" xml:"put-code"`
		Status  string   `json:"status" xml:"status"`
	}

	writeResponse(w, r, UpdateResponse{PutCode: putCode, Status: "updated"})
}

// Helper structs for employment
type GenericEmploymentResponse struct {
	XMLName        xml.Name `json:"-" xml:"employment:employment"`
	PutCode        int      `json:"put-code" xml:"put-code"`
	DepartmentName string   `json:"department-name" xml:"department-name"`
	RoleTitle      string   `json:"role-title" xml:"role-title"`
	Organization   Org      `json:"organization" xml:"organization"`
	StartDate      DateYear `json:"start-date" xml:"start-date"`
}

func handleGetEmployment(w http.ResponseWriter, r *http.Request) {
	putCode, _ := strconv.Atoi(r.PathValue("putCode"))

	response := GenericEmploymentResponse{
		PutCode:        putCode,
		DepartmentName: "Mock Department",
		RoleTitle:      "Mock Researcher",
		Organization:   Org{Name: "Mock Org"},
		StartDate: DateYear{
			Year: Value{Value: "2020"},
		},
	}
	writeResponse(w, r, response)
}

func handlePostEmployment(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	newPutCode := rand.Intn(999999) + 100000

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/employment/%d", orcid, newPutCode))
	w.WriteHeader(http.StatusCreated)

	type PutCodeResponse struct {
		XMLName xml.Name `json:"-" xml:"response"`
		PutCode int      `json:"put-code" xml:"put-code"`
	}

	writeResponse(w, r, PutCodeResponse{PutCode: newPutCode})
}

func handlePutEmployment(w http.ResponseWriter, r *http.Request) {
	orcid := r.PathValue("orcid")
	putCode := r.PathValue("putCode")

	w.Header().Set("Location", fmt.Sprintf("https://api.orcid.org/v3.0/%s/employment/%s", orcid, putCode))
	w.WriteHeader(http.StatusOK)

	type UpdateResponse struct {
		XMLName xml.Name `json:"-" xml:"response"`
		PutCode string   `json:"put-code" xml:"put-code"`
		Status  string   `json:"status" xml:"status"`
	}

	writeResponse(w, r, UpdateResponse{PutCode: putCode, Status: "updated"})
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

	writeResponse(w, r, resp)
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
