package main

import (
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-chi/chi/v5"
	"github.com/idkwhyureadthis/ozon-task/graph"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/database"
	"github.com/idkwhyureadthis/ozon-task/internal/pkg/mw"
	"github.com/joho/godotenv"
)

const defaultPort = "8080"

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("failed to load .env file")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	storage := os.Getenv("STORAGE")
	migrations := os.Getenv("MIGRATIONS")
	database.Connect(storage, migrations)
	graph.Init()
	defer database.CloseDB()

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	router := chi.NewRouter()
	router.Handle("/", playground.Handler("GraphQL playground", "/query"))

	authGroup := router.Group(nil)
	authGroup.Use(mw.AuthMiddleware)
	authGroup.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
