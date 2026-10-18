// Package main implements a small user-search service together with a test
// harness for its client.
//
// SearchServer plays the role of an "external system": it reads users from an
// XML dataset and serves them over HTTP with filtering, sorting and pagination.
// SearchClient (see client.go) is the consumer of that API.
package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

// datasetPath is the path to the XML file with users. It is a package-level
// variable so that tests can point the server at a different (or broken) file.
var datasetPath = "dataset.xml"

// validAccessToken is the only value of the AccessToken header that
// SearchServer accepts. Any other value results in 401 Unauthorized.
const validAccessToken = "good_token"

// errBadOrderBy is returned by parseParams when the order_by parameter is not
// a number or is not one of OrderByAsc, OrderByAsIs and OrderByDesc.
var errBadOrderBy = errors.New("OrderBy invalid")

// xmlRow mirrors a single <row> element of the dataset. Only the fields that
// are needed to build a User are declared; all other elements are ignored.
type xmlRow struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Gender    string `xml:"gender"`
	About     string `xml:"about"`
}

// xmlRoot mirrors the <root> element of the dataset and holds all its rows.
type xmlRoot struct {
	Rows []xmlRow `xml:"row"`
}

// searchParams holds validated query parameters of a search request.
type searchParams struct {
	// limit is the maximum number of users to return.
	limit int
	// offset is the number of users to skip after filtering and sorting.
	offset int
	// query is a substring to look for in the Name and About fields.
	// An empty query matches every user.
	query string
	// orderField is the normalized sort field: "id", "age" or "name".
	orderField string
	// orderBy is the sort direction: OrderByAsc, OrderByAsIs or OrderByDesc.
	orderBy int
}

// parseParams extracts and validates the search parameters from the URL query
// of r: limit, offset, query, order_field and order_by.
//
// The limit and offset parameters must be non-negative integers, order_by must
// be one of OrderByAsc, OrderByAsIs and OrderByDesc, and order_field must be
// empty or one of "id", "age" and "name" (case-insensitive). An empty
// order_field defaults to sorting by name.
//
// On failure the returned error describes the first invalid parameter; for an
// unknown order_field its text equals ErrorBadOrderField.
func parseParams(r *http.Request) (searchParams, error) {
	q := r.URL.Query()
	p := searchParams{query: q.Get("query")}

	var err error
	if p.limit, err = strconv.Atoi(q.Get("limit")); err != nil || p.limit < 0 {
		return p, errors.New("limit invalid")
	}
	if p.offset, err = strconv.Atoi(q.Get("offset")); err != nil || p.offset < 0 {
		return p, errors.New("offset invalid")
	}
	if p.orderBy, err = strconv.Atoi(q.Get("order_by")); err != nil ||
		(p.orderBy != OrderByAsc && p.orderBy != OrderByAsIs && p.orderBy != OrderByDesc) {
		return p, errBadOrderBy
	}

	switch strings.ToLower(q.Get("order_field")) {
	case "", "name":
		p.orderField = "name"
	case "id":
		p.orderField = "id"
	case "age":
		p.orderField = "age"
	default:
		return p, errors.New(ErrorBadOrderField)
	}
	return p, nil
}

// loadUsers reads the XML dataset at path and converts every row into a User.
// The user's Name is built as the first name followed by a space and the last
// name. The file is parsed with a streaming xml.Decoder.
//
// It returns an error if the file cannot be opened or contains malformed XML.
func loadUsers(path string) ([]User, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var root xmlRoot
	if err := xml.NewDecoder(f).Decode(&root); err != nil {
		return nil, err
	}

	users := make([]User, 0, len(root.Rows))
	for _, row := range root.Rows {
		users = append(users, User{
			ID:     row.ID,
			Name:   row.FirstName + " " + row.LastName,
			Age:    row.Age,
			About:  row.About,
			Gender: row.Gender,
		})
	}
	return users, nil
}

// filterUsers returns the users whose Name or About contains query as a
// case-sensitive substring. An empty query returns users unchanged.
func filterUsers(users []User, query string) []User {
	if query == "" {
		return users
	}
	res := make([]User, 0)
	for _, u := range users {
		if strings.Contains(u.Name, query) || strings.Contains(u.About, query) {
			res = append(res, u)
		}
	}
	return res
}

// sortUsers sorts users in place by the given field ("id", "age"; anything
// else sorts by name) in the direction given by orderBy. With OrderByAsIs the
// slice is left untouched. The sort is stable, so users with equal keys keep
// their original relative order.
func sortUsers(users []User, field string, orderBy int) {
	if orderBy == OrderByAsIs {
		return
	}
	less := func(i, j int) bool {
		switch field {
		case "id":
			return users[i].ID < users[j].ID
		case "age":
			return users[i].Age < users[j].Age
		default:
			return users[i].Name < users[j].Name
		}
	}
	if orderBy == OrderByDesc {
		asc := less
		less = func(i, j int) bool { return asc(j, i) }
	}
	sort.SliceStable(users, less)
}

// paginate returns at most limit users starting at index offset. It returns an
// empty, non-nil slice when offset is past the end of users.
func paginate(users []User, offset, limit int) []User {
	if offset >= len(users) {
		return []User{}
	}
	end := offset + limit
	if end > len(users) {
		end = len(users)
	}
	return users[offset:end]
}

// sendJSON marshals v to JSON and writes it to w with the given HTTP status
// and a JSON content type. If v cannot be marshaled, it responds with 500
// Internal Server Error and no body. A failure to write the response is logged.
func sendJSON(w http.ResponseWriter, status int, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if _, err := io.WriteString(w, string(data)); err != nil {
		log.Printf("write response error: %v", err)
	}
}

// SearchServer is an http.HandlerFunc that searches users in the XML dataset
// located at datasetPath. It is the server side of SearchClient.
//
// The request must carry the AccessToken header equal to validAccessToken and
// the following query parameters:
//
//	limit        maximum number of users to return
//	offset       number of users to skip after filtering and sorting
//	query        substring to look for in Name or About (empty: no filtering)
//	order_field  "id", "age" or "name" (empty: "name")
//	order_by     1 ascending, 0 keep file order, -1 descending
//
// Responses:
//
//	200  a JSON array of users
//	400  a JSON SearchErrorResponse describing the invalid parameter
//	401  the access token is missing or wrong
//	500  the dataset cannot be read or parsed
func SearchServer(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("AccessToken") != validAccessToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	params, err := parseParams(r)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, SearchErrorResponse{Error: err.Error()})
		return
	}

	users, err := loadUsers(datasetPath)
	if err != nil {
		log.Printf("load dataset error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	users = filterUsers(users, params.query)
	sortUsers(users, params.orderField, params.orderBy)
	users = paginate(users, params.offset, params.limit)

	sendJSON(w, http.StatusOK, users)
}
