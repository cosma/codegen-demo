package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cosma/codegen-demo/generated/api"
	"github.com/cosma/codegen-demo/generated/db"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/lib/pq"
)

type Server struct {
	db  *db.Queries
	api *api.Server
}

func NewServer(db *db.Queries) *Server {
	return &Server{
		db:  db,
		api: api.NewServer(nil, nil),
	}
}

func (s *Server) ListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.db.ListTasks(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert DB tasks to API tasks
	apiTasks := make([]api.Task, len(tasks))
	for i, task := range tasks {
		apiTasks[i] = api.Task{
			Id:          task.ID,
			Title:       task.Title,
			Description: task.Description,
			Completed:   task.Completed,
			CreatedAt:   task.CreatedAt,
		}
	}

	api.WriteJSONResponse(w, http.StatusOK, apiTasks)
}

func (s *Server) CreateTask(w http.ResponseWriter, r *http.Request) {
	var req api.CreateTaskRequest
	if err := api.ReadJSONRequest(r, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task, err := s.db.CreateTask(r.Context(), db.CreateTaskParams{
		Title:       req.Title,
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	apiTask := api.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description.String,
		Completed:   task.Completed,
		CreatedAt:   task.CreatedAt,
	}

	api.WriteJSONResponse(w, http.StatusCreated, apiTask)
}

func (s *Server) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "taskId")
	if taskID == "" {
		http.Error(w, "task ID is required", http.StatusBadRequest)
		return
	}

	task, err := s.db.GetTask(r.Context(), taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "task not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	apiTask := api.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description.String,
		Completed:   task.Completed,
		CreatedAt:   task.CreatedAt,
	}

	api.WriteJSONResponse(w, http.StatusOK, apiTask)
}

func main() {
	// Get database configuration from environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "codegen_demo")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")

	// Construct database connection string
	dbConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode,
	)

	// Connect to database
	dbConn, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	// Wait for database to be ready
	for i := 0; i < 5; i++ {
		err = dbConn.Ping()
		if err == nil {
			break
		}
		log.Printf("Waiting for database to be ready... (attempt %d/5)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Could not connect to database:", err)
	}

	// Create queries instance
	queries := db.New(dbConn)

	// Create server
	server := NewServer(queries)

	// Create router
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Register routes
	r.Get("/tasks", server.ListTasks)
	r.Post("/tasks", server.CreateTask)
	r.Get("/tasks/{taskId}", server.GetTask)

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start server
	go func() {
		log.Printf("Server starting on :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
