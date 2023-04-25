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

// 封裝請求
type ShortenRequest struct {
	URL string `json:"url"`
}

// 回應數據
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

	//啟動HTTP服務器
	http.ListenAndServe(":8080", r)
}

// 調用輸入模板
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

// 短網址處理
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

	// 將長網址及短網址存入Redis 中
	err = redisClient.Set(r.Context(), fmt.Sprintf("%x", shortID), req.URL, 0).Err()
	if err != nil {
		http.Error(w, "Error saving URL to Redis", http.StatusInternalServerError)
		return
	}

	shortURL := fmt.Sprintf("http://localhost:8080/%x", shortID)

	//回應短網址
	response := ShortenResponse{ShortURL: shortURL}
	json.NewEncoder(w).Encode(response)
}

// 重定向處理，瀏覽器訪問短網址，從Redis查詢相對應的長網址，並返回給瀏覽器
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

// 生成短網址的哈希函數，將長網址轉成短網址
func generateShortID(url string) (uint64, error) {
	h := fnv.New64a()
	_, err := h.Write([]byte(url))
	if err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}
