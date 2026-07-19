package main

import (
	"context"
	"diabetify/database"
	"diabetify/docs"
	"diabetify/internal/cache"
	"diabetify/internal/config"
	"diabetify/internal/controllers"
	"diabetify/internal/middleware"
	"diabetify/internal/ml"
	"diabetify/internal/repository"
	"diabetify/internal/services"
	"diabetify/routes"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	loadEnv()
	settings := config.Load()

	if err := settings.ValidateRuntime(); err != nil {
		log.Fatal(err)
	}

	// Check if sharding is enabled
	useSharding := settings.UseSharding
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
		plannerRepo           repository.PlannerRepository
		auditLogRepo          repository.AuditLogRepository
	)

	if useSharding {
		// Use sharded repositories
		forgotPasswordRepo = repository.NewResetPasswordRepository(nil)
		userRepo = repository.NewUserRepository(nil)
		verificationRepo = repository.NewVerificationRepository(nil)
		activityRepo = repository.NewShardedActivityRepository()
		articleRepo = repository.NewArticleRepository(database.DB)
		profileRepo = repository.NewShardedUserProfileRepository()
		predictionRepo = repository.NewShardedPredictionRepository()
		predictionJobRepo = repository.NewShardedPredictionJobRepository()
		counterfactualJobRepo = repository.NewCounterfactualJobRepository(nil)
		plannerRepo = repository.NewShardedPlannerRepository()
		auditLogRepo = repository.NewAuditLogRepository(database.DB)
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
		predictionJobRepo = repository.NewPredictionJobRepository(database.DB)
		counterfactualJobRepo = repository.NewCounterfactualJobRepository(database.DB)
		plannerRepo = repository.NewPlannerRepository(database.DB)
		auditLogRepo = repository.NewAuditLogRepository(database.DB)
		log.Println("Initialized single database repositories")
	}

	rabbitMQURL := settings.RabbitMQ.URL

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
	workerCount := runtime.NumCPU() * 8
	if workerCount < 20 {
		workerCount = 20
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
	plannerService := services.NewPlannerService(plannerRepo, profileRepo, activityRepo)

	// Initialize controllers
	userController := controllers.NewUserController(userService)
	verificationController := controllers.NewVerificationController(verificationService)
	oauthController := controllers.NewOauthController(oauthService)
	activityController := controllers.NewActivityController(activityService)
	articleController := controllers.NewArticleController(articleService)
	profileController := controllers.NewUserProfileController(profileService)
	plannerController := controllers.NewPlannerController(plannerService)

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

	router.GET("/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Service is alive",
		})
	})

	router.GET("/health/ready", func(c *gin.Context) {
		statusCode, payload := buildReadinessPayload(c.Request.Context(), predictionJobWorker, counterfactualJobService)
		c.JSON(statusCode, payload)
	})

	auditMiddleware := middleware.AuditMiddleware(auditLogRepo)
	auditLogController := controllers.NewAuditLogController(auditLogRepo, userRepo)

	routes.RegisterUserRoutes(router, userController, auditMiddleware)
	routes.RegisterVerificationRoutes(router, verificationController)
	routes.RegisterSwaggerRoutes(router)
	routes.RegisterOauthRoutes(router, oauthController)
	routes.RegisterActivityRoutes(router, activityController)
	routes.RegisterArticleRoutes(router, articleController)
	routes.RegisterUserProfileRoutes(router, profileController)
	routes.RegisterPredictionRoutes(router, predictionController)
	routes.RegisterCounterfactualRoutes(router, counterfactualController)
	routes.RegisterPlannerRoutes(router, plannerController)
	routes.RegisterDataRoutes(router, userRepo, auditMiddleware)
	routes.RegisterExpertRoutes(router, userRepo, auditMiddleware)
	routes.RegisterAuditLogRoutes(router, auditLogController)

	// Debug endpoints
	debug := router.Group("/debug")
	debug.GET("/stats", func(c *gin.Context) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		c.JSON(200, gin.H{
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  m.Alloc / 1024 / 1024,
			"workers":    workerCount,
			"job_worker": predictionJobWorker.GetStatus(),
		})
	})

	debug.GET("/jobs", func(c *gin.Context) {
		c.JSON(200, predictionJobWorker.GetStatus())
	})

	debug.GET("/counterfactual", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"counterfactual_service": counterfactualJobService.GetStatus(),
		})
	})

	if useSharding {
		debug.GET("/shards", func(c *gin.Context) {
			shardsHealth := database.CheckShardsHealth()
			c.JSON(200, gin.H{
				"shards_health": shardsHealth,
				"total_shards":  len(shardsHealth),
			})
		})
	} else {
		debug.GET("/database", func(c *gin.Context) {
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
	port := settings.Port

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

	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != nil {
			log.Fatal("Failed to start server:", err)
		}
	case sig := <-shutdownSignals:
		log.Printf("Received signal %s, shutting down server...", sig)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		}
		log.Println("HTTP server stopped")
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

func buildReadinessPayload(
	ctx context.Context,
	predictionJobWorker services.PredictionJobWorker,
	counterfactualJobService services.CounterfactualJobService,
) (int, gin.H) {
	checks := gin.H{
		"database":       checkDatabase(ctx),
		"prediction_job": predictionJobWorker.GetStatus(),
		"counterfactual": counterfactualJobService.GetStatus(),
		"redis":          checkRedis(),
	}

	ready := true
	if dbCheck, ok := checks["database"].(gin.H); ok && dbCheck["ready"] != true {
		ready = false
	}
	if workerStatus, ok := checks["prediction_job"].(map[string]interface{}); ok {
		if workerStatus["running"] != true || workerStatus["rabbitmq_connected"] != true {
			ready = false
		}
	}
	if cfStatus, ok := checks["counterfactual"].(map[string]interface{}); ok {
		if cfStatus["running"] != true || cfStatus["rabbitmq_connected"] != true {
			ready = false
		}
	}
	if redisCheck, ok := checks["redis"].(gin.H); ok && redisCheck["ready"] != true {
		ready = false
	}

	status := "success"
	message := "Service is ready"
	statusCode := http.StatusOK
	if !ready {
		status = "error"
		message = "Service is not ready"
		statusCode = http.StatusServiceUnavailable
	}

	return statusCode, gin.H{
		"status":  status,
		"message": message,
		"checks":  checks,
	}
}

func checkDatabase(ctx context.Context) gin.H {
	sqlDB, err := database.DB.DB()
	if err != nil {
		return gin.H{"ready": false, "error": err.Error()}
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return gin.H{"ready": false, "error": err.Error()}
	}

	return gin.H{"ready": true}
}

func checkRedis() gin.H {
	client, err := cache.NewRedisClient()
	if err != nil {
		return gin.H{"ready": false, "error": err.Error()}
	}
	defer client.Close()

	return gin.H{"ready": true}
}
