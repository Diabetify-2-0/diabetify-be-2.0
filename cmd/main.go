package main

import (
	"context"
	"diabetify/database"
	"diabetify/docs"
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"
	"diabetify/internal/ml"
	"diabetify/internal/repository"
	"diabetify/internal/services"
	"diabetify/routes"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()

	// Check if sharding is enabled
	useSharding := os.Getenv("USE_SHARDING") == "true"
	log.Printf("Sharding mode: %v", useSharding)

	// Swagger Documentation
	docs.SwaggerInfo.Title = "Diabetify API"
	docs.SwaggerInfo.Description = "This is api of Diabetify App with Async ML Service via RabbitMQ."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Connect to database based on sharding configuration
	if useSharding {
		database.ConnectShardedDatabase()
		if err := database.MigrateAllShards(); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
		database.MonitorShardedDBConnections()
	} else {
		database.ConnectDatabase()
		if err := database.MigrateDatabase(); err != nil {
			log.Fatalf("Failed to run database migrations: %v", err)
		}
		database.MonitorDBConnections()
	}

	// Initialize repositories based on sharding configuration
	var (
		forgotPasswordRepo    repository.ResetPasswordRepository
		userRepo              repository.UserRepository
		verificationRepo      repository.VerificationRepository
		activityRepo          repository.ActivityRepository
		articleRepo           repository.ArticleRepository
		profileRepo           repository.UserProfileRepository
		predictionRepo        repository.PredictionRepository
		predictionJobRepo     repository.PredictionJobRepository
		counterfactualJobRepo repository.CounterfactualJobRepository
	)

	predictionJobRepo = repository.NewPredictionJobRepository(database.DB)
	counterfactualJobRepo = repository.NewCounterfactualJobRepository(database.DB)

	if useSharding {
		// Use sharded repositories
		forgotPasswordRepo = repository.NewResetPasswordRepository(nil)
		userRepo = repository.NewUserRepository(nil)
		verificationRepo = repository.NewVerificationRepository(nil)
		activityRepo = repository.NewShardedActivityRepository()
		articleRepo = repository.NewArticleRepository(database.DB)
		profileRepo = repository.NewShardedUserProfileRepository()
		predictionRepo = repository.NewShardedPredictionRepository()
		log.Println("Initialized sharded repositories")
	} else {
		// Use single database repositories
		forgotPasswordRepo = repository.NewResetPasswordRepository(database.DB)
		userRepo = repository.NewUserRepository(database.DB)
		verificationRepo = repository.NewVerificationRepository(database.DB)
		activityRepo = repository.NewActivityRepository(database.DB)
		articleRepo = repository.NewArticleRepository(database.DB)
		profileRepo = repository.NewUserProfileRepository(database.DB)
		predictionRepo = repository.NewPredictionRepository(database.DB)
		log.Println("Initialized single database repositories")
	}

	rabbitMQURL := getEnv("RABBITMQ_URL", "amqp://admin:password123@localhost:5672/")

	log.Printf("Connecting to ML service via RabbitMQ: %s", rabbitMQURL)

	mlClient, err := ml.NewAsyncMLClient(
		rabbitMQURL,
		"ml.prediction.response",
	)
	if err != nil {
		log.Fatal("Failed to create ML RabbitMQ client:", err)
	}
	defer mlClient.Close()

	// Test ML service connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mlClient.HealthCheckAsync(ctx); err != nil {
		log.Printf("Warning: ML service health check failed: %v", err)
		log.Println("The application will start, but predictions will fail until ML service is available")
	} else {
		log.Println("ML service connection established successfully")
	}

	// Initialize Prediction Job Worker
	workerCount := runtime.NumCPU()
	if workerCount < 3 {
		workerCount = 3
	}

	predictionJobWorker := services.NewPredictionJobWorker(
		predictionJobRepo,
		predictionRepo,
		userRepo,
		profileRepo,
		activityRepo,
		mlClient,
		workerCount,
		rabbitMQURL,
	)

	log.Printf("Starting prediction job worker with %d workers...", workerCount)
	predictionJobWorker.Start()
	defer predictionJobWorker.Stop()

	counterfactualJobService := services.NewCounterfactualJobService(
		counterfactualJobRepo,
		rabbitMQURL,
		"ml.cf.request",
		"ml.cf.response",
	)
	if err := counterfactualJobService.Start(); err != nil {
		log.Fatalf("Failed to start counterfactual job service: %v", err)
	}
	defer counterfactualJobService.Stop()

	// Initialize services
	userService := services.NewUserService(userRepo, forgotPasswordRepo)
	profileService := services.NewUserProfileService(profileRepo)
	verificationService := services.NewVerificationService(verificationRepo, userRepo)
	activityService := services.NewActivityService(activityRepo)
	articleService := services.NewArticleService(articleRepo)
	oauthService := services.NewOAuthService(userRepo)

	// Initialize controllers
	userController := controllers.NewUserController(userService)
	verificationController := controllers.NewVerificationController(verificationService)
	oauthController := controllers.NewOauthController(oauthService)
	activityController := controllers.NewActivityController(activityService)
	articleController := controllers.NewArticleController(articleService)
	profileController := controllers.NewUserProfileController(profileService)

	// UNIFIED Prediction Controller (handles both sync and async via job worker)
	predictionController := controllers.NewPredictionController(
		predictionRepo,
		userRepo,
		profileRepo,
		activityRepo,
		predictionJobRepo,   // Job repository
		predictionJobWorker, // Job worker
		mlClient,            // ML client for health checks
	)
	counterfactualController := controllers.NewCounterfactualController(
		counterfactualJobRepo,
		counterfactualJobService,
	)

	gin.SetMode(gin.ReleaseMode)
	// Setup Gin router
	router := gin.Default()

	router.Use(middleware.CORSMiddleware())

	router.GET("/", func(c *gin.Context) {
		response := gin.H{
			"message":        "Diabetify API is running",
			"version":        "1.0.0",
			"status":         "healthy",
			"ml_service":     "RabbitMQ async worker",
			"prediction":     "Async prediction jobs via RabbitMQ",
			"counterfactual": "Async counterfactual jobs via RabbitMQ",
		}

		if useSharding {
			response["database"] = "Sharded PostgreSQL"
			response["shards"] = []string{"shard1 (users 1-5000)", "shard2 (users 5001-10000)"}
		} else {
			response["database"] = "Single PostgreSQL"
		}

		c.JSON(200, response)
	})

	routes.RegisterUserRoutes(router, userController)
	routes.RegisterVerificationRoutes(router, verificationController)
	routes.RegisterSwaggerRoutes(router)
	routes.RegisterOauthRoutes(router, oauthController)
	routes.RegisterActivityRoutes(router, activityController)
	routes.RegisterArticleRoutes(router, articleController)
	routes.RegisterUserProfileRoutes(router, profileController)
	routes.RegisterPredictionRoutes(router, predictionController)
	routes.RegisterCounterfactualRoutes(router, counterfactualController)
	routes.RegisterDataRoutes(router, userRepo)
	routes.RegisterExpertRoutes(router)

	// Debug endpoints
	router.GET("/debug/stats", func(c *gin.Context) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		c.JSON(200, gin.H{
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  m.Alloc / 1024 / 1024,
			"workers":    workerCount,
			"job_worker": predictionJobWorker.GetStatus(),
		})
	})

	// Job worker specific debug endpoint
	router.GET("/debug/jobs", func(c *gin.Context) {
		c.JSON(200, predictionJobWorker.GetStatus())
	})

	router.GET("/debug/counterfactual", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"counterfactual_service": counterfactualJobService.GetStatus(),
		})
	})

	// Conditional shard health check endpoint
	if useSharding {
		router.GET("/debug/shards", func(c *gin.Context) {
			shardsHealth := database.CheckShardsHealth()
			c.JSON(200, gin.H{
				"shards_health": shardsHealth,
				"total_shards":  len(shardsHealth),
			})
		})
	} else {
		router.GET("/debug/database", func(c *gin.Context) {
			// Simple database health check for single DB
			sqlDB, err := database.DB.DB()
			if err != nil {
				c.JSON(500, gin.H{
					"database_health": false,
					"mode":            "single_database",
					"error":           err.Error(),
				})
				return
			}

			var result int
			row := sqlDB.QueryRowContext(c.Request.Context(), "SELECT 1")
			err = row.Scan(&result)
			isHealthy := err == nil && result == 1

			c.JSON(200, gin.H{
				"database_health": isHealthy,
				"mode":            "single_database",
			})
		})
	}

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Printf("API Documentation: http://localhost:%s/swagger/index.html", port)
	log.Printf("Health Check: http://localhost:%s/prediction/health", port)

	if useSharding {
		log.Printf("Database Shards Health: http://localhost:%s/debug/shards", port)
	} else {
		log.Printf("Database Health: http://localhost:%s/debug/database", port)
	}

	log.Printf("Job Worker Debug: http://localhost:%s/debug/jobs", port)

	server := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	runtime.GOMAXPROCS(runtime.NumCPU())

	log.Printf("Diabetify API Server started successfully on port %s", port)
	log.Printf("Using %d CPU cores for job processing", workerCount)
	log.Printf("Async prediction jobs ready via RabbitMQ")

	if err := server.ListenAndServe(); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func loadEnv() {
	if err := godotenv.Load(".env"); err == nil {
		return
	}
	if err := godotenv.Load("../.env"); err == nil {
		return
	}
	log.Println("No .env file found; using process environment")
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
