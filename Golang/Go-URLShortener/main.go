package main

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"html/template"
	"net/http"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
)

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

var redisClient *redis.Client

func main() {
	// 初始化 Redis 連接
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	r := mux.NewRouter()

	// 路由和處理函數
	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/shorten", shortenHandler).Methods("POST")
	r.HandleFunc("/{shortID}", redirectHandler).Methods("GET")

	http.ListenAndServe(":8080", r)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("home.html")
	if err != nil {
		http.Error(w, "Error parsing home template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing home template", http.StatusInternalServerError)
		return
	}
}

func shortenHandler(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	shortID, err := generateShortID(req.URL)
	if err != nil {
		http.Error(w, "Error generating short ID", http.StatusInternalServerError)
		return
	}

	// 将长网址和短网址绑定存入 Redis 中
	err = redisClient.Set(r.Context(), fmt.Sprintf("%x", shortID), req.URL, 0).Err()
	if err != nil {
		http.Error(w, "Error saving URL to Redis", http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("http://localhost:8080/%x", shortID)

	response := ShortenResponse{ShortURL: shortURL}
	json.NewEncoder(w).Encode(response)
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortID := vars["shortID"]

	longURL, err := redisClient.Get(r.Context(), shortID).Result()
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, longURL, http.StatusSeeOther)
}

func generateShortID(url string) (uint64, error) {
	h := fnv.New64a()
	_, err := h.Write([]byte(url))
	if err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}
