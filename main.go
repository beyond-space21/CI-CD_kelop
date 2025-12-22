package main

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"log"
	"net/http"
	"os"
	_"fmt"

	"time"

	"github.com/joho/godotenv"

	Event "hifi/Events"
	Auth "hifi/Services/Auth"
	ES "hifi/Services/Elasticsearch"
	Mdb "hifi/Services/Mdb"
	Utils "hifi/Utils"
	Storage "hifi/Services/Storage"
)

var ServerPort string

func loadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
		os.Exit(1)
	}

	ServerPort = ":" + os.Getenv("GO_SERVER_PORT")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timestamp := time.Now().Format("2006-01-02 15:04:05")

		fmt.Printf("[%s] %s %s\n", timestamp, r.Method, r.URL.Path)

		next.ServeHTTP(w, r)
	})
}

func main() {
	loadEnv()
	Mdb.InitPostgres()
	
	// Run migrations if RUN_MIGRATIONS env var is set
	if os.Getenv("RUN_MIGRATIONS") == "true" {
		fmt.Println("Running database migrations...")
		if err := Mdb.RunMigrations(); err != nil {
			log.Fatal("Migration failed:", err)
		}
		fmt.Println("Migrations completed successfully")
	}
	
	Auth.Initauth()
	Utils.InitEnv()
	Utils.InitEnv()
	Storage.InitStorage()
	ES.InitElasticsearch()
	Event.Init()
	
	mux := chi.NewRouter()
	mux.Use(corsMiddleware,loggingMiddleware)
	Event.Handler(mux)

	mux.Handle("/webStatic/", http.StripPrefix("/webStatic/", http.FileServer(http.Dir("webStatic"))))

	fmt.Println("Server started at " + ServerPort)
	err := http.ListenAndServe(ServerPort, mux)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
