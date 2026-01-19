package main

import (
	"html/template"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
	"github.com/nuuner/spines/internal/config"
	"github.com/nuuner/spines/internal/database"
	"github.com/nuuner/spines/internal/handlers"
	"github.com/nuuner/spines/internal/middleware"
	"github.com/nuuner/spines/internal/models"
)

func main() {
	cfg := config.Load()

	if cfg.AdminPassword == "" {
		log.Fatal("ADMIN_PASSWORD environment variable is required")
	}

	if err := database.Connect(cfg.DatabasePath); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Clean up expired sessions on startup
	if err := models.DeleteExpiredSessions(); err != nil {
		log.Printf("Warning: Failed to clean up expired sessions: %v", err)
	} else {
		log.Println("Cleaned up expired sessions")
	}

	engine := html.New("./web/templates", ".html")
	engine.AddFuncMap(template.FuncMap{
		"subtract": func(a, b int) int {
			return a - b
		},
	})

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Static("/static", "./web/static")

	// Security headers middleware
	app.Use(middleware.SecurityHeaders)

	// CSRF protection middleware
	app.Use(middleware.CSRFProtection)

	// Navigation context middleware - runs on all routes
	app.Use(middleware.NavContext)

	authHandler := handlers.NewAuthHandler(cfg)
	booksHandler := handlers.NewBooksHandler(cfg)
	userBooksHandler := handlers.NewUserBooksHandler(cfg)

	// Public routes
	app.Get("/", handlers.Dashboard)
	app.Get("/api/images/book/:id", handlers.ProxyBookCover)
	app.Get("/api/events", handlers.GetLatestEvents)
	app.Get("/api/events/recent", handlers.GetRecentEvents)
	app.Get("/api/events/user/:username", handlers.GetUserEvents)
	app.Get("/u/:username", handlers.UserPage)
	app.Get("/u/:username/shelf/:shelf", handlers.GetPublicShelfBooks)

	// User auth routes (with rate limiting on login)
	app.Get("/login", handlers.UserLoginPage)
	app.Post("/login", middleware.RateLimitAuth, handlers.UserLogin)
	app.Post("/logout", handlers.UserLogout)

	// User routes (protected) - profile
	app.Get("/profile", middleware.UserAuth, handlers.ProfilePage)
	app.Post("/profile", middleware.UserAuth, handlers.UpdateProfile)
	app.Post("/profile/password", middleware.UserAuth, handlers.ChangePassword)
	app.Post("/profile/avatar", middleware.UserAuth, handlers.UploadAvatar)
	app.Post("/profile/avatar/remove", middleware.UserAuth, handlers.RemoveAvatar)
	app.Post("/profile/theme", middleware.UserAuth, handlers.UpdateTheme)

	// User routes (protected) - my books
	myBooks := app.Group("/my-books", middleware.UserAuth)
	myBooks.Get("/", userBooksHandler.MyBooks)
	myBooks.Get("/search", userBooksHandler.SearchBooks)
	myBooks.Get("/add", userBooksHandler.AddBookPage)
	myBooks.Get("/shelf/:shelf", userBooksHandler.GetShelfBooks)
	myBooks.Post("/", userBooksHandler.AddBook)
	myBooks.Post("/:book_id", userBooksHandler.UpdateBook)
	myBooks.Post("/:book_id/dates", userBooksHandler.UpdateBookDates)
	myBooks.Post("/:book_id/delete", userBooksHandler.RemoveBook)

	// Admin auth routes (with rate limiting on login)
	app.Get("/admin/login", authHandler.LoginPage)
	app.Post("/admin/login", middleware.RateLimitAuth, authHandler.Login)
	app.Post("/admin/logout", authHandler.Logout)

	// Admin routes (protected)
	admin := app.Group("/admin", middleware.AdminAuth)
	admin.Get("/", handlers.AdminDashboard)
	admin.Get("/users", handlers.AdminUsersList)
	admin.Post("/users", handlers.AdminCreateUser)
	admin.Get("/users/:id/edit", handlers.AdminEditUser)
	admin.Post("/users/:id", handlers.AdminUpdateUser)
	admin.Post("/users/:id/delete", handlers.AdminDeleteUser)
	admin.Post("/users/:id/password", handlers.AdminSetPassword)
	admin.Post("/users/:id/password/clear", handlers.AdminClearPassword)
	admin.Get("/users/:id/books", booksHandler.ManageUserBooks)
	admin.Get("/users/:id/books/search", booksHandler.SearchBooks)
	admin.Post("/users/:id/books", booksHandler.AddBook)
	admin.Post("/users/:id/books/:book_id", booksHandler.UpdateBook)
	admin.Post("/users/:id/books/:book_id/delete", booksHandler.RemoveBook)

	log.Printf("Starting server on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
