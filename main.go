package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/leiper-mike/chirpy/internal/auth"
	"github.com/leiper-mike/chirpy/internal/database"
	_ "github.com/lib/pq"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
type LoggedInUser struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}
type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}
type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	platform       string
	secret         string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}
func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	dbQueries := database.New(db)
	apiCfg := apiConfig{fileserverHits: atomic.Int32{}, dbQueries: dbQueries, platform: os.Getenv("PLATFORM"), secret: os.Getenv("TOKEN_SECRET")}
	serveMux := http.NewServeMux()
	fileHandler := http.FileServer(http.Dir("./app"))
	serveMux.Handle("/app/", http.StripPrefix("/app", apiCfg.middlewareMetricsInc(fileHandler)))
	serveMux.HandleFunc("GET /api/healthz", readyHandler)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.countHandler)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.reset)
	serveMux.HandleFunc("POST /api/users", apiCfg.addUserHandler)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.postChirpHandler)
	serveMux.HandleFunc("POST /api/login", apiCfg.loginHandler)
	serveMux.HandleFunc("POST /api/refresh", apiCfg.refreshHandler)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.revokeHandler)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.getAllChirpsHandler)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.getChirpHandler)
	server := http.Server{Addr: ":8080", Handler: serveMux}
	server.ListenAndServe()
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) countHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html")
	w.WriteHeader(200)
	str := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(str))
}

func (cfg *apiConfig) reset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(403)
		w.Write([]byte("Forbidden"))
		return
	} else {
		cfg.fileserverHits.Store(0)
		err := cfg.dbQueries.DeleteUsers(context.Background())
		if err != nil {
			w.Header().Add("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(500)
			w.Write([]byte("Internal server error"))
		} else {
			w.Header().Add("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		}
	}

}

func cleanBody(str string) string {
	words := strings.Split(str, " ")
	new := []string{}
	for _, word := range words {
		upWord := strings.ToUpper(word)
		if upWord == "KERFUFFLE" || upWord == "SHARBERT" || upWord == "FORNAX" {
			new = append(new, "****")
		} else {
			new = append(new, word)
		}
	}

	return strings.Join(new, " ")
}
func (cfg *apiConfig) addUserHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Print("Error decoding request")
		w.WriteHeader(500)
		return
	}
	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		fmt.Print(err.Error())
		w.WriteHeader(500)
		return
	}
	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: hash})
	if err != nil {
		fmt.Print(err.Error())
		w.WriteHeader(500)
		return
	}
	user := User{ID: dbUser.ID, CreatedAt: dbUser.CreatedAt.Time, UpdatedAt: dbUser.UpdatedAt.Time, Email: dbUser.Email}
	dat, err := json.Marshal(user)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	w.Write(dat)

}

func (cfg *apiConfig) postChirpHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}
	type errVals struct {
		Error string `json:"error"`
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}
	userId, err := auth.ValidateJWT(token, cfg.secret)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(401)
		return
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Error decoding parameters: %s", err)
		errvals := errVals{Error: err.Error()}
		dat, err := json.Marshal(errvals)
		if err != nil {
			fmt.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		w.Write(dat)
		return
	}
	if len(params.Body) <= 140 {
		cleanedBody := cleanBody(params.Body)
		dbChirp, err := cfg.dbQueries.CreateChirp(context.Background(), database.CreateChirpParams{Body: params.Body, UserID: userId})
		if err != nil {
			fmt.Printf("Error creating chirp:%v\n", err.Error())
			w.WriteHeader(500)
			return
		}
		chirp := Chirp{ID: dbChirp.ID, CreatedAt: dbChirp.CreatedAt.Time, UpdatedAt: dbChirp.UpdatedAt.Time, Body: cleanedBody, UserID: dbChirp.UserID}
		dat, err := json.Marshal(chirp)
		if err != nil {
			fmt.Printf("Error marshalling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(201)
		w.Write(dat)
		return
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(400)
		w.Write([]byte("Chirp length must be no greater than 140 characters"))
	}
}
func (cfg *apiConfig) getAllChirpsHandler(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.dbQueries.GetAllChirps(context.Background())
	if err != nil {
		fmt.Printf("Error getting chirps:%v\n", err.Error())
		w.WriteHeader(500)
		return
	}
	chirps := make([]Chirp, 10)
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, Chirp{ID: dbChirp.ID, CreatedAt: dbChirp.CreatedAt.Time, UpdatedAt: dbChirp.UpdatedAt.Time, Body: dbChirp.Body, UserID: dbChirp.UserID})
	}
	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
	})
	dat, err := json.Marshal(chirps)
	if err != nil {
		fmt.Print(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
func (cfg *apiConfig) getChirpHandler(w http.ResponseWriter, r *http.Request) {
	chirpId := r.PathValue("chirpID")
	uuid, err := uuid.Parse(chirpId)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(500)
		return
	}
	dbChirp, err := cfg.dbQueries.GetChirp(context.Background(), uuid)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			w.WriteHeader(404)
			return
		}
		fmt.Println(err.Error())
		w.WriteHeader(500)
		return
	}
	dat, err := json.Marshal(Chirp{ID: dbChirp.ID, CreatedAt: dbChirp.CreatedAt.Time, UpdatedAt: dbChirp.UpdatedAt.Time, Body: dbChirp.Body, UserID: dbChirp.UserID})
	if err != nil {

		fmt.Print(err.Error())
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
func (cfg *apiConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		fmt.Printf("Error decoding request: %v", err)
		w.WriteHeader(500)
		return
	}
	dbUser, err := cfg.dbQueries.GetUserByEmail(context.Background(), params.Email)
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			w.WriteHeader(404)
			return
		}
		fmt.Println(err.Error())
		w.WriteHeader(500)
		return
	}
	err = auth.CheckPasswordHash(params.Password, dbUser.HashedPassword)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	token, err := auth.MakeJWT(dbUser.ID, cfg.secret)
	if err != nil {
		fmt.Printf("Error creating token: %v", err)
		w.WriteHeader(500)
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		fmt.Printf("Error creating refresh token: %v", err)
		w.WriteHeader(500)
		return
	}
	dur, _ := time.ParseDuration("1440h")
	exp := time.Now().UTC().Add(dur)
	_, err = cfg.dbQueries.CreateRefreshToken(context.Background(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: dbUser.ID, ExpiresAt: exp})
	if err != nil {
		fmt.Printf("Error inserting refresh token: %v", err)
		w.WriteHeader(500)
		return
	}
	user := LoggedInUser{ID: dbUser.ID, CreatedAt: dbUser.CreatedAt.Time, UpdatedAt: dbUser.UpdatedAt.Time, Email: dbUser.Email, Token: token, RefreshToken: refreshToken}
	dat, err := json.Marshal(user)
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
func (cfg *apiConfig) refreshHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	rToken, err := cfg.dbQueries.GetRefreshTokenByID(context.Background(), token)
	if err != nil {
		fmt.Println(err.Error())
		w.WriteHeader(401)
		return
	}
	if rToken.RevokedAt.Valid{
		fmt.Println("Refresh token has been revoked")
		w.WriteHeader(401)
		return
	}
	if !time.Now().Before(rToken.ExpiresAt){
		fmt.Println("Refresh token has expired")
		w.WriteHeader(401)
		return
	}
	newToken, err := auth.MakeJWT(rToken.UserID, cfg.secret)
	if err != nil {
		fmt.Printf("Error creating token: %v", err)
		w.WriteHeader(500)
		return
	}
	type ret struct{
		Token string `json:"token"`
	}
	dat, err := json.Marshal(ret{Token: newToken})
	if err != nil {
		fmt.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
}
func (cfg *apiConfig) revokeHandler(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	err = cfg.dbQueries.RevokeToken(context.Background(), token)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	w.WriteHeader(204)
}
