package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"music-server/auth"
	"music-server/config"
	"music-server/database"
	"music-server/enrichment"
	"music-server/handlers"
	"music-server/router"
	. "music-server/scanner"
	"music-server/utils"
	"music-server/websocket"
)

// @title Music Server API
// @version 2.0
// @description A RESTful music server with authentication, playlist management, and real-time WebSocket support
// @host localhost:8080
// @BasePath /api
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	log.Printf("Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded successfully. Music path: %s", cfg.MusicPath)

	// Initialize database
	dbConfig := database.Config{
		ConnectionString: cfg.DatabaseURL,
	}
	db, err := database.NewDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if configuredArtworkPath, settingErr := db.GetSetting(database.ArtworkPathSettingKey); settingErr != nil {
		log.Printf("Warning: Failed to load artwork storage setting: %v", settingErr)
	} else if configuredArtworkPath != "" && os.Getenv("WAVENODE_ARTWORK_PATH") == "" {
		if err := os.Setenv("WAVENODE_ARTWORK_PATH", configuredArtworkPath); err != nil {
			log.Printf("Warning: Failed to apply artwork storage setting: %v", err)
		} else {
			log.Printf("Using configured artwork storage: %s", utils.ArtworkDirectory())
		}
	}
	if migrated, migrationErr := utils.MigrateLegacyArtwork(); migrationErr != nil {
		log.Printf("Warning: Failed to migrate legacy artwork: %v", migrationErr)
	} else if migrated > 0 {
		log.Printf("Migrated %d artwork files into persistent storage", migrated)
	}
	if repaired, repairErr := db.RepairMissingArtistLinks(); repairErr != nil {
		log.Printf("Warning: Failed to repair track artist links: %v", repairErr)
	} else if repaired > 0 {
		log.Printf("Repaired artist links for %d tracks", repaired)
	}

	if err := db.ImportLegacyMusicSource(cfg.MusicPath); err != nil {
		log.Printf("Warning: Failed to import legacy music path: %v", err)
	}

	// Initialize auth handler
	authHandler := auth.NewAuthHandler(db, []byte(cfg.JWTSecret))
	authHandler.SetRegistrationEnabled(cfg.AllowRegistration)

	// Initialize WebSocket manager
	wsManager := websocket.NewWebSocketManager(authHandler)
	wsManager.Start()
	websocket.SetGlobalWebSocketManager(wsManager)

	// Initialize scanner
	log.Printf("Initializing scanner with database-managed music sources")
	scannerInstance := NewScanner(db, "")
	log.Printf("Scanner initialized successfully")
	autoUpdateContext, stopAutoUpdates := context.WithCancel(context.Background())
	defer stopAutoUpdates()
	autoUpdater := NewAutoUpdater(db, scannerInstance)
	autoUpdater.Start(autoUpdateContext)

	// Make scanner available to handlers
	handlers.InitScanner(scannerInstance)
	log.Printf("Scanner made available to handlers")

	// Initialize enrichment scanner
	scanStore := database.NewScanStore(db)
	enrichmentScanner := enrichment.NewEnrichmentScanner(db, scanStore)
	log.Printf("Enrichment scanner initialized successfully")

	// Initialize handlers
	musicHandler := handlers.NewMusicHandler(db)
	playlistHandler := handlers.NewPlaylistHandler(db)

	// Initialize CORS configuration
	corsConfig := struct {
		AllowedOrigins []string `json:"allowed_origins"`
		AllowedMethods []string `json:"allowed_methods"`
		AllowedHeaders []string `json:"allowed_headers"`
	}{
		AllowedOrigins: cfg.CORSOrigins,
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders: []string{
			"Content-Type",
			"Authorization",
			"X-Requested-With",
			"Accept",
			"Origin",
			"Access-Control-Request-Method",
			"Access-Control-Request-Headers",
		},
	}

	// Initialize router
	appRouter := router.NewRouter(
		authHandler,
		musicHandler,
		playlistHandler,
		wsManager,
		db,
		corsConfig,
	)

	// Set enrichment scanner on the router's enrichment handler
	appRouter.SetEnrichmentScanner(enrichmentScanner)
	appRouter.SetAutoUpdater(autoUpdater)
	log.Printf("Enrichment scanner set on router")

	// Setup routes
	httpRouter := appRouter.SetupRoutes()

	// Create HTTP server
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpRouter,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// Streaming responses must not be capped by a whole-response timeout.
		WriteTimeout: 0,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting server on port %s", cfg.Port)
		log.Printf("API Documentation: http://localhost:%s/swagger/index.html", cfg.Port)
		log.Printf("Health Check: http://localhost:%s/health", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
