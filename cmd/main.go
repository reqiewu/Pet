package main

import (
	"PetStore/internal/db"
	"PetStore/internal/handler"
	"PetStore/transport"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	_ "PetStore/docs" // Импорт документации Swagger
	"PetStore/internal/repository"
	"PetStore/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func init() {
	godotenv.Load()
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

// @title PetStore API
// @version 1.0
// @description API для управления питомцами и заказами
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
func main() {
	// Загрузка конфигурации
	ctx := context.Background()
	transport := transport.JSONResponder{}
	// Инициализация БД
	dbPool, err := db.NewPostgresConnection(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// Инициализация слоёв
	userRepo := repository.NewUserStore(dbPool)
	petRepo := repository.NewStore(dbPool)
	storeRepo := repository.NewStoreRepository(dbPool)

	userService := service.NewUserService(userRepo)
	petService := service.NewPetService(petRepo)
	storeService := service.NewStoreService(storeRepo)

	userHandler := handler.NewUserHandler(userService, transport)
	petHandler := handler.NewPetHandler(petService)
	storeHandler := handler.NewStoreHandler(storeService)

	// Создание роутера chi
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Public routes (не требуют аутентификации)
	r.Group(func(r chi.Router) {
		// User routes
		r.Post("/user", userHandler.CreateUser)
		r.Post("/user/createWithArray", userHandler.CreateUsersWithArray)
		r.Post("/user/createWithList", userHandler.CreateUsersWithList)
		r.Get("/user/login", userHandler.LoginUser)
		r.Get("/user/logout", userHandler.LogoutUser)
		r.Get("/user/{username}", userHandler.GetUserByName)

		// Store routes
		r.Get("/store/inventory", storeHandler.GetInventory)
		r.Post("/store/order", storeHandler.PlaceOrder)
		r.Get("/store/order/{orderId}", storeHandler.GetOrderById)
		r.Delete("/store/order/{orderId}", storeHandler.DeleteOrder)
	})

	// Protected routes (требуют JWT)
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(jwtauth.New("HS256", []byte(os.Getenv("JWT_SECRET")), nil)))
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token, _, _ := jwtauth.FromContext(r.Context())
				if token == nil {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		// User routes
		r.Put("/user/{username}", userHandler.UpdateUser)
		r.Delete("/user/{username}", userHandler.DeleteUser)

		// Pet routes
		r.Post("/pet", petHandler.AddPet)
		r.Put("/pet", petHandler.UpdatePet)
		r.Get("/pet/findByStatus", petHandler.FindPetsByStatus)
		r.Get("/pet/{petId}", petHandler.GetPetById)
		r.Post("/pet/{petId}", petHandler.UpdatePetWithForm)
		r.Delete("/pet/{petId}", petHandler.DeletePet)
		r.Post("/pet/{petId}/uploadImage", petHandler.UploadImage)
	})

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	// HTTP сервер
	srv := &http.Server{
		Addr:         ":" + os.Getenv("PORT"),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()
	log.Printf("Server started on %s", srv.Addr)

	<-done
	log.Println("Server stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}
