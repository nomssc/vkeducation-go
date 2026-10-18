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

// datasetPath — путь до файла с данными (глобальная переменная, чтобы тесты могли его подменять).
var datasetPath = "dataset.xml"

// validAccessToken — токен, который SearchServer считает корректным.
const validAccessToken = "good_token"

var errBadOrderBy = errors.New("OrderBy invalid")

type xmlRow struct {
	ID        int    `xml:"id"`
	Age       int    `xml:"age"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Gender    string `xml:"gender"`
	About     string `xml:"about"`
}

type xmlRoot struct {
	Rows []xmlRow `xml:"row"`
}

type searchParams struct {
	limit      int
	offset     int
	query      string
	orderField string // нормализованное: "id", "age" или "name"
	orderBy    int
}

// parseParams разбирает и валидирует параметры запроса.
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

// loadUsers читает пользователей из xml-файла потоково через Decoder.
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

// sendJSON сериализует v и отправляет клиенту с нужным статусом.
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

// SearchServer — «внешняя система», ищущая пользователей в dataset.xml.
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
