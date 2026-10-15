package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/faust8888/go-musthave-metrics/internal/handler"
	"github.com/faust8888/go-musthave-metrics/internal/repository"
)

func exampleRouter() http.Handler {
	store := repository.NewMemStorage()
	r := chi.NewRouter()
	r.Get("/", handler.Index(store))
	r.Post("/update/{type}/{name}/{value}", handler.Update(store, nil))
	r.Get("/value/{type}/{name}", handler.Value(store))
	r.Post("/update", handler.UpdateJSON(store, nil))
	r.Post("/update/", handler.UpdateJSON(store, nil))
	r.Post("/value", handler.ValueJSON(store))
	r.Post("/value/", handler.ValueJSON(store))
	r.Post("/updates/", handler.UpdatesBatch(store, nil))
	return r
}

func ExampleUpdate() {
	ts := httptest.NewServer(exampleRouter())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/update/gauge/Alloc/123.4", "text/plain", nil)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

func ExampleValue() {
	ts := httptest.NewServer(exampleRouter())
	defer ts.Close()

	_, _ = http.Post(ts.URL+"/update/counter/PollCount/7", "text/plain", nil)
	resp, err := http.Get(ts.URL + "/value/counter/PollCount")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode, string(body))
	// Output:
	// 200 7
}

func ExampleUpdateJSON() {
	ts := httptest.NewServer(exampleRouter())
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/update", "application/json",
		strings.NewReader(`{"id":"Alloc","type":"gauge","value":123.4}`))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":123.4}
}

func ExampleValueJSON() {
	ts := httptest.NewServer(exampleRouter())
	defer ts.Close()

	_, _ = http.Post(ts.URL+"/update", "application/json",
		strings.NewReader(`{"id":"Alloc","type":"gauge","value":42}`))
	resp, err := http.Post(ts.URL+"/value", "application/json",
		strings.NewReader(`{"id":"Alloc","type":"gauge"}`))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(resp.StatusCode)
	fmt.Println(string(body))
	// Output:
	// 200
	// {"id":"Alloc","type":"gauge","value":42}
}

func ExampleUpdatesBatch() {
	ts := httptest.NewServer(exampleRouter())
	defer ts.Close()

	payload := `[
		{"id":"Alloc","type":"gauge","value":1.5},
		{"id":"PollCount","type":"counter","delta":3}
	]`
	resp, err := http.Post(ts.URL+"/updates/", "application/json", strings.NewReader(payload))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	// Output:
	// 200
}

func ExampleIndex() {
	ts := httptest.NewServer(exampleRouter())
	defer ts.Close()

	_, _ = http.Post(ts.URL+"/update/gauge/Alloc/1", "text/plain", nil)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	fmt.Println(resp.StatusCode)
	fmt.Println(resp.Header.Get("Content-Type"))
	// Output:
	// 200
	// text/html; charset=utf-8
}
