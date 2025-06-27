package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)
type apiConfig struct{
	fileserverHits atomic.Int32
}
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func main(){
	apiCfg := apiConfig{fileserverHits: atomic.Int32{}}
	serveMux := http.NewServeMux()
	fileHandler := http.FileServer(http.Dir("./app"))
	serveMux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(fileHandler)))
	serveMux.HandleFunc("/", readyHandler)
	serveMux.HandleFunc("/metrics", apiCfg.countHandler)
	serveMux.HandleFunc("/reset", apiCfg.reset)
	server := http.Server{Addr: ":8080",Handler: serveMux}
	server.ListenAndServe()
}

func readyHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) countHandler(w http.ResponseWriter, r *http.Request){
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	str := fmt.Sprintf("Hits: %d", cfg.fileserverHits.Load())
	w.Write([]byte(str))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request){
	cfg.fileserverHits.Store(0)
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}
