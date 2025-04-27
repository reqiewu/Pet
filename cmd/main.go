package main

import (
	"PetStore/internal/db"
	"PetStore/internal/handler"
	"PetStore/transport"
	"context"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"PetStore/internal/repository"
	"PetStore/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/jwtauth/v5"
)

func init() {
	godotenv.Load()
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}
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
		r.Post("/api/user", userHandler.CreateUser)
		r.Get("/api/user/createWithArray", userHandler.CreateUsersWithArray)
		r.Post("/api/user/createWithList", userHandler.CreateUsersWithList)
		r.Get("/api/user/login", userHandler.LoginUser)
		r.Get("/api/user/logout", userHandler.LogoutUser)
		r.Get("/api/user/{username}", userHandler.GetUserByName)
		r.Post("/api/store/order", storeHandler.PlaceOrder)
		r.Get("/api/store/order/{orderId}", storeHandler.GetOrderById)
		r.Delete("/api/store/order/{orderId}", storeHandler.DeleteOrder)
	})

	// Protected routes (требуют JWT)
	r.Group(func(r chi.Router) {
		r.Use(jwtauth.Verifier(jwtauth.New("HS256", []byte(os.Getenv("JWT_SECRET")), nil)))
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				token, _, _ := jwtauth.FromContext(r.Context())
				if token == nil {
					http.Error(w, "Forbidden", http.StatusForbidden) // 403
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		// User routes
		r.Put("/api/user/{username}", userHandler.UpdateUser)
		r.Delete("/api/user/{username}", userHandler.DeleteUser)

		// Pet routes
		r.Post("/api/pet", petHandler.AddPet)
		r.Put("/api/pet", petHandler.UpdatePet)
		r.Get("/api/pet/findByStatus", petHandler.FindPetsByStatus)
		r.Get("/api/pet/{petId}", petHandler.GetPetById)
		r.Post("/api/pet/{petId}", petHandler.UpdatePetWithForm)
		r.Delete("/api/pet/{petId}", petHandler.DeletePet)
		r.Post("/api/pet/{petId}", petHandler.UploadImage)
		// Store routes
		r.Get("/api/store/inventory", storeHandler.GetInventory)
	})

	// Swagger UI (если используется)
	r.Handle("/swagger/*", http.StripPrefix("/swagger/", http.FileServer(http.Dir("./swagger"))))

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
