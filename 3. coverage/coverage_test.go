package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// newSearchClient starts a test HTTP server backed by SearchServer and returns
// a SearchClient that talks to it using the given access token. The server is
// shut down automatically when the test finishes.
func newSearchClient(t *testing.T, token string) *SearchClient {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	t.Cleanup(ts.Close)
	return &SearchClient{AccessToken: token, URL: ts.URL}
}

// withDataset points SearchServer at the dataset file at path for the duration
// of the test and restores the previous path afterwards.
func withDataset(t *testing.T, path string) {
	t.Helper()
	old := datasetPath
	datasetPath = path
	t.Cleanup(func() { datasetPath = old })
}

// userNames returns the Name of every user, preserving order.
func userNames(users []User) []string {
	res := make([]string, 0, len(users))
	for _, u := range users {
		res = append(res, u.Name)
	}
	return res
}

// userIDs returns the ID of every user, preserving order.
func userIDs(users []User) []int {
	res := make([]int, 0, len(users))
	for _, u := range users {
		res = append(res, u.ID)
	}
	return res
}

// idRange returns the consecutive integers from..to inclusive.
func idRange(from, to int) []int {
	res := make([]int, 0, to-from+1)
	for i := from; i <= to; i++ {
		res = append(res, i)
	}
	return res
}

// TestFindUsersSuccess runs SearchClient.FindUsers against the real
// SearchServer and the real dataset, and checks the exact users, their order
// and the NextPage flag for sorting, searching and pagination scenarios.
func TestFindUsersSuccess(t *testing.T) {
	cl := newSearchClient(t, validAccessToken)

	cases := []struct {
		name     string
		req      SearchRequest
		wantIDs  []int // nil means "do not check"
		wantName []string
		wantNext bool
	}{
		{
			name:     "default order by name asc",
			req:      SearchRequest{Limit: 3, OrderBy: OrderByAsc},
			wantName: []string{"Allison Valdez", "Annie Osborn", "Bell Bauer"},
			wantNext: true,
		},
		{
			name:     "name desc",
			req:      SearchRequest{Limit: 3, OrderField: "Name", OrderBy: OrderByDesc},
			wantName: []string{"Whitley Davidson", "Twila Snow", "Terrell Hall"},
			wantNext: true,
		},
		{
			name:     "as is keeps file order",
			req:      SearchRequest{Limit: 3, OrderBy: OrderByAsIs},
			wantIDs:  []int{0, 1, 2},
			wantName: []string{"Boyd Wolf", "Hilda Mayer", "Brooks Aguilar"},
			wantNext: true,
		},
		{
			name:     "id desc",
			req:      SearchRequest{Limit: 2, OrderField: "Id", OrderBy: OrderByDesc},
			wantIDs:  []int{34, 33},
			wantName: []string{"Kane Sharp", "Twila Snow"},
			wantNext: true,
		},
		{
			name:     "age asc (stable for equal ages)",
			req:      SearchRequest{Limit: 3, OrderField: "Age", OrderBy: OrderByAsc},
			wantIDs:  []int{1, 15, 23},
			wantNext: true,
		},
		{
			name:     "readme example: oldest user with 'on'",
			req:      SearchRequest{Limit: 1, Offset: 0, Query: "on", OrderField: "age", OrderBy: OrderByDesc},
			wantIDs:  []int{13},
			wantName: []string{"Whitley Davidson"},
			wantNext: true,
		},
		{
			name:     "query by name",
			req:      SearchRequest{Limit: 10, Query: "Boyd", OrderBy: OrderByAsIs},
			wantIDs:  []int{0},
			wantName: []string{"Boyd Wolf"},
			wantNext: false,
		},
		{
			name:     "query by about only",
			req:      SearchRequest{Limit: 10, Query: "Sit commodo consectetur minim", OrderBy: OrderByAsIs},
			wantIDs:  []int{1},
			wantName: []string{"Hilda Mayer"},
			wantNext: false,
		},
		{
			name:     "query without matches",
			req:      SearchRequest{Limit: 10, Query: "zzzzzz-nothing", OrderBy: OrderByAsc},
			wantIDs:  []int{},
			wantNext: false,
		},
		{
			name:     "limit capped to 25, more pages exist",
			req:      SearchRequest{Limit: 100, OrderBy: OrderByAsIs},
			wantIDs:  idRange(0, 24),
			wantNext: true,
		},
		{
			name:     "second page, last",
			req:      SearchRequest{Limit: 25, Offset: 25, OrderBy: OrderByAsIs},
			wantIDs:  idRange(25, 34),
			wantNext: false,
		},
		{
			name:     "offset inside data, exact end",
			req:      SearchRequest{Limit: 5, Offset: 30, OrderBy: OrderByAsIs},
			wantIDs:  idRange(30, 34),
			wantNext: false,
		},
		{
			name:     "offset beyond data",
			req:      SearchRequest{Limit: 5, Offset: 100, OrderBy: OrderByAsIs},
			wantIDs:  []int{},
			wantNext: false,
		},
		{
			name:     "limit zero gives no users",
			req:      SearchRequest{Limit: 0, OrderBy: OrderByAsIs},
			wantIDs:  []int{},
			wantNext: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := cl.FindUsers(tc.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.NextPage != tc.wantNext {
				t.Errorf("NextPage = %v, want %v", resp.NextPage, tc.wantNext)
			}
			if tc.wantIDs != nil && !reflect.DeepEqual(userIDs(resp.Users), tc.wantIDs) {
				t.Errorf("ids = %v, want %v", userIDs(resp.Users), tc.wantIDs)
			}
			if tc.wantName != nil && !reflect.DeepEqual(userNames(resp.Users), tc.wantName) {
				t.Errorf("names = %v, want %v", userNames(resp.Users), tc.wantName)
			}
		})
	}
}

// TestFindUsersReturnsFullRecord checks that every User field, not just the
// ID and Name, is filled in correctly from the dataset.
func TestFindUsersReturnsFullRecord(t *testing.T) {
	cl := newSearchClient(t, validAccessToken)
	resp, err := cl.FindUsers(SearchRequest{Limit: 1, Query: "Boyd", OrderBy: OrderByAsIs})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Users) != 1 {
		t.Fatalf("got %d users, want 1", len(resp.Users))
	}
	u := resp.Users[0]
	if u.ID != 0 || u.Name != "Boyd Wolf" || u.Age != 22 || u.Gender != "male" {
		t.Errorf("unexpected user: %+v", u)
	}
	if !strings.HasPrefix(u.About, "Nulla cillum enim voluptate") {
		t.Errorf("unexpected About: %q", u.About)
	}
}

// TestFindUsersErrors checks the error returned by FindUsers for invalid
// requests: those rejected by the client itself and those rejected by
// SearchServer with 400 or 401. On error the response must be nil.
func TestFindUsersErrors(t *testing.T) {
	cases := []struct {
		name    string
		token   string
		req     SearchRequest
		wantErr string
	}{
		{"negative limit", validAccessToken, SearchRequest{Limit: -1}, "limit must be > 0"},
		{"negative offset", validAccessToken, SearchRequest{Limit: 1, Offset: -1}, "offset must be > 0"},
		{"bad token", "wrong", SearchRequest{Limit: 1}, "bad AccessToken"},
		{"bad order field", validAccessToken, SearchRequest{Limit: 1, OrderField: "Gender"}, "OrderFeld Gender invalid"},
		{"bad order by", validAccessToken, SearchRequest{Limit: 1, OrderBy: 5}, "unknown bad request error: OrderBy invalid"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cl := newSearchClient(t, tc.token)
			resp, err := cl.FindUsers(tc.req)
			if err == nil {
				t.Fatalf("expected error %q, got response %+v", tc.wantErr, resp)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tc.wantErr)
			}
			if resp != nil {
				t.Errorf("response must be nil on error, got %+v", resp)
			}
		})
	}
}

// TestFindUsersBrokenExternalSystem points FindUsers at fake servers that
// misbehave (500, malformed JSON, slow response) and checks that the client
// reports each failure with the expected error.
func TestFindUsersBrokenExternalSystem(t *testing.T) {
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantSubstr string
	}{
		{
			name: "500 from server",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantSubstr: "SearchServer fatal error",
		},
		{
			name: "400 with broken json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("not a json"))
			},
			wantSubstr: "cant unpack error json",
		},
		{
			name: "200 with broken json",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte("not a json"))
			},
			wantSubstr: "cant unpack result json",
		},
		{
			name: "timeout",
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(client.Timeout + 300*time.Millisecond)
			},
			wantSubstr: "timeout for",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(tc.handler)
			defer ts.Close()
			cl := &SearchClient{AccessToken: validAccessToken, URL: ts.URL}
			_, err := cl.FindUsers(SearchRequest{Limit: 1})
			if err == nil || !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Errorf("error = %v, want containing %q", err, tc.wantSubstr)
			}
		})
	}
}

// TestFindUsersUnknownError checks that a transport error that is not a
// timeout (here: an unsupported URL scheme) is reported as an unknown error.
func TestFindUsersUnknownError(t *testing.T) {
	cl := &SearchClient{AccessToken: validAccessToken, URL: "unsupported-scheme://host"}
	_, err := cl.FindUsers(SearchRequest{Limit: 1})
	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("error = %v, want containing %q", err, "unknown error")
	}
}

// TestFindUsersSendsAccessTokenAndParams verifies the wire format of the
// request produced by FindUsers: the AccessToken header and all query
// parameters, including the limit incremented by one to detect a next page.
func TestFindUsersSendsAccessTokenAndParams(t *testing.T) {
	var gotToken string
	var gotQuery map[string][]string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("AccessToken")
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte("[]"))
	}))
	defer ts.Close()

	cl := &SearchClient{AccessToken: "tok", URL: ts.URL}
	_, err := cl.FindUsers(SearchRequest{Limit: 5, Offset: 2, Query: "q", OrderField: "Age", OrderBy: OrderByDesc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotToken != "tok" {
		t.Errorf("token = %q, want tok", gotToken)
	}
	want := map[string][]string{
		"limit":       {"6"}, // the client asks for one extra record
		"offset":      {"2"},
		"query":       {"q"},
		"order_field": {"Age"},
		"order_by":    {"-1"},
	}
	if !reflect.DeepEqual(gotQuery, want) {
		t.Errorf("query = %v, want %v", gotQuery, want)
	}
}

// callServer invokes SearchServer in-process with the given raw URL query and
// access token (omitted when empty) and returns the recorded response.
func callServer(rawQuery, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/?"+rawQuery, nil)
	if token != "" {
		req.Header.Set("AccessToken", token)
	}
	rec := httptest.NewRecorder()
	SearchServer(rec, req)
	return rec
}

// TestSearchServerBadParams checks that malformed or out-of-range query
// parameters are rejected with 400 and a descriptive error message.
func TestSearchServerBadParams(t *testing.T) {
	cases := []struct {
		name      string
		rawQuery  string
		wantError string
	}{
		{"limit is not a number", "limit=abc&offset=0&order_by=0", "limit invalid"},
		{"limit is missing", "offset=0&order_by=0", "limit invalid"},
		{"limit is negative", "limit=-1&offset=0&order_by=0", "limit invalid"},
		{"offset is not a number", "limit=1&offset=abc&order_by=0", "offset invalid"},
		{"offset is negative", "limit=1&offset=-3&order_by=0", "offset invalid"},
		{"order_by is not a number", "limit=1&offset=0&order_by=abc", "OrderBy invalid"},
		{"order_by is out of range", "limit=1&offset=0&order_by=2", "OrderBy invalid"},
		{"unknown order_field", "limit=1&offset=0&order_by=0&order_field=about", ErrorBadOrderField},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := callServer(tc.rawQuery, validAccessToken)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.wantError) {
				t.Errorf("body = %q, want containing %q", rec.Body.String(), tc.wantError)
			}
		})
	}
}

// TestSearchServerUnauthorized checks that a missing or wrong access token
// results in 401 Unauthorized.
func TestSearchServerUnauthorized(t *testing.T) {
	for _, token := range []string{"", "wrong"} {
		rec := callServer("limit=1&offset=0&order_by=0", token)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("token %q: status = %d, want 401", token, rec.Code)
		}
	}
}

// TestSearchServerOrderFieldCaseInsensitiveAndOK checks that every supported
// order_field value, in any letter case, is accepted, and that an empty value
// is accepted too.
func TestSearchServerOrderFieldCaseInsensitiveAndOK(t *testing.T) {
	for _, f := range []string{"", "id", "ID", "Age", "age", "Name", "name"} {
		rec := callServer("limit=1&offset=0&order_by=1&order_field="+f, validAccessToken)
		if rec.Code != http.StatusOK {
			t.Errorf("order_field %q: status = %d, want 200", f, rec.Code)
		}
	}
}

// TestSearchServerEmptyResultIsJSONArray checks that a search without matches
// is encoded as an empty JSON array rather than null.
func TestSearchServerEmptyResultIsJSONArray(t *testing.T) {
	rec := callServer("limit=5&offset=0&order_by=0&query=zzzzzz-nothing", validAccessToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

// TestSearchServerDatasetErrors checks that an unreadable or malformed dataset
// file leads to 500 Internal Server Error, which the client reports as a
// fatal server error.
func TestSearchServerDatasetErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		withDataset(t, filepath.Join(t.TempDir(), "no-such-file.xml"))
		rec := callServer("limit=1&offset=0&order_by=0", validAccessToken)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rec.Code)
		}
	})

	t.Run("broken xml", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "broken.xml")
		if err := os.WriteFile(path, []byte("<root><row><id>1</id>"), 0o600); err != nil {
			t.Fatal(err)
		}
		withDataset(t, path)
		rec := callServer("limit=1&offset=0&order_by=0", validAccessToken)
		if rec.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want 500", rec.Code)
		}
	})

	t.Run("client sees fatal error", func(t *testing.T) {
		withDataset(t, filepath.Join(t.TempDir(), "no-such-file.xml"))
		cl := newSearchClient(t, validAccessToken)
		_, err := cl.FindUsers(SearchRequest{Limit: 1})
		if err == nil || err.Error() != "SearchServer fatal error" {
			t.Errorf("error = %v, want SearchServer fatal error", err)
		}
	})
}

// TestSearchServerCustomDataset swaps the dataset for a tiny hand-written one
// and checks that the whole User record, including the joined first and last
// name, is mapped and sorted correctly.
func TestSearchServerCustomDataset(t *testing.T) {
	const data = `<?xml version="1.0" encoding="UTF-8" ?>
<root>
  <row><id>7</id><age>50</age><first_name>Ivan</first_name><last_name>Petrov</last_name><gender>male</gender><about>likes go</about></row>
  <row><id>3</id><age>20</age><first_name>Anna</first_name><last_name>Ivanova</last_name><gender>female</gender><about>likes xml</about></row>
</root>`
	path := filepath.Join(t.TempDir(), "mini.xml")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	withDataset(t, path)

	cl := newSearchClient(t, validAccessToken)
	resp, err := cl.FindUsers(SearchRequest{Limit: 10, OrderField: "Id", OrderBy: OrderByAsc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []User{
		{ID: 3, Name: "Anna Ivanova", Age: 20, About: "likes xml", Gender: "female"},
		{ID: 7, Name: "Ivan Petrov", Age: 50, About: "likes go", Gender: "male"},
	}
	if !reflect.DeepEqual(resp.Users, want) {
		t.Errorf("users = %+v, want %+v", resp.Users, want)
	}
	if resp.NextPage {
		t.Error("NextPage must be false")
	}
}

// failingWriter is an http.ResponseWriter whose Write always fails. It records
// the status code passed to WriteHeader.
type failingWriter struct {
	header http.Header
	status int
}

// Header returns the header map of the failing writer.
func (f *failingWriter) Header() http.Header { return f.header }

// WriteHeader records the status code.
func (f *failingWriter) WriteHeader(status int) { f.status = status }

// Write always fails with an error.
func (f *failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

// TestSendJSONMarshalError checks that a value that cannot be marshaled to
// JSON (a channel) yields 500 Internal Server Error.
func TestSendJSONMarshalError(t *testing.T) {
	rec := httptest.NewRecorder()
	sendJSON(rec, http.StatusOK, make(chan int)) // channels cannot be encoded as JSON
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
}

// TestSendJSONWriteError checks that sendJSON survives a failing writer: the
// status is still sent and the write error is only logged.
func TestSendJSONWriteError(t *testing.T) {
	w := &failingWriter{header: http.Header{}}
	sendJSON(w, http.StatusOK, []User{})
	if w.status != http.StatusOK {
		t.Errorf("status = %d, want 200", w.status)
	}
}
