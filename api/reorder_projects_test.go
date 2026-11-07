package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jakeloud/jl/entities"
)

func TestReorderProjectsAPI(t *testing.T) {
	originalConfFile := entities.CONF_FILE
	entities.CONF_FILE = filepath.Join(t.TempDir(), "conf.json")
	t.Cleanup(func() { entities.CONF_FILE = originalConfFile })

	if err := entities.SetConf(entities.Config{
		Projects: []entities.Project{
			{Name: entities.JAKELOUD, Port: 666},
			{Name: "first", Repo: "https://example.com/first", Domain: []string{"first.example.com"}, Additional: map[string]interface{}{"cmd": []string{"run first"}}},
			{Name: "second", Port: 8080, State: "running"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := entities.SetUser("owner@example.com", "password"); err != nil {
		t.Fatal(err)
	}
	initial, err := entities.GetConf()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		additional map[string]interface{}
		password   string
		wantError  bool
	}{
		{name: "saves requested order", additional: map[string]interface{}{"order": []string{"second", "first"}}, password: "password"},
		{name: "missing additional", password: "password", wantError: true},
		{name: "missing order", additional: map[string]interface{}{}, password: "password", wantError: true},
		{name: "order is not an array", additional: map[string]interface{}{"order": "second,first"}, password: "password", wantError: true},
		{name: "null order", additional: map[string]interface{}{"order": nil}, password: "password", wantError: true},
		{name: "non-string project", additional: map[string]interface{}{"order": []interface{}{"second", 1}}, password: "password", wantError: true},
		{name: "unknown project", additional: map[string]interface{}{"order": []string{"second", "unknown"}}, password: "password", wantError: true},
		{name: "missing project", additional: map[string]interface{}{"order": []string{"second"}}, password: "password", wantError: true},
		{name: "empty order", additional: map[string]interface{}{"order": []string{}}, password: "password", wantError: true},
		{name: "duplicate replaces project", additional: map[string]interface{}{"order": []string{"second", "second"}}, password: "password", wantError: true},
		{name: "system project included", additional: map[string]interface{}{"order": []string{entities.JAKELOUD, "first", "second"}}, password: "password", wantError: true},
		{name: "unauthenticated", additional: map[string]interface{}{"order": []string{"second", "first"}}, wantError: true},
		{name: "wrong password", additional: map[string]interface{}{"order": []string{"second", "first"}}, password: "wrong", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := entities.SetConf(initial); err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(map[string]interface{}{
				"op":         "reorderProjectsOp",
				"email":      "owner@example.com",
				"password":   tt.password,
				"additional": tt.additional,
			})
			if err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			API(response, httptest.NewRequest(http.MethodPost, "/api", bytes.NewReader(body)))
			if tt.wantError {
				if response.Code < http.StatusBadRequest {
					t.Fatalf("expected an error response, got %d", response.Code)
				}
			} else if response.Code != http.StatusOK {
				t.Fatalf("expected success, got %d: %s", response.Code, response.Body.String())
			}

			actual, err := entities.GetConf()
			if err != nil {
				t.Fatal(err)
			}
			want := initial
			if !tt.wantError {
				want.Projects = []entities.Project{initial.Projects[0], initial.Projects[2], initial.Projects[1]}
			}
			if !reflect.DeepEqual(actual, want) {
				t.Fatalf("saved configuration differs from expected: got %+v, want %+v", actual, want)
			}
		})
	}
}
