package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"naivepanel/internal/auth"
	"naivepanel/internal/config"
	"naivepanel/internal/database"
	"naivepanel/internal/handlers"
	"naivepanel/internal/models"
	"naivepanel/internal/naiveproxy"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

//go:embed web/*
var webFS embed.FS

func main() {
	configPath := flag.String("config", "", "Path to config file")
	dataDir := flag.String("data-dir", "", "Data directory")
	port := flag.Int("port", 0, "Panel port (overrides config)")
	setup := flag.Bool("setup", false, "Initialize database and print credentials, then exit")
	flag.Parse()

	// Determine config path
	cfgPath := *configPath
	if cfgPath == "" {
		dd := *dataDir
		if dd == "" {
			dd = config.DefaultDataDir()
		}
		cfgPath = filepath.Join(dd, "config.json")
	}

	// Load configuration
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *dataDir != "" {
		cfg.DataDir = *dataDir
	}
	if *port != 0 {
		cfg.PanelPort = *port
	}

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.DBPath())
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create admin if not exists
	if _, err := db.GetAdmin(); err != nil {
		adminUser := config.GenerateRandomUsername(8)
		adminPass := config.GenerateRandomPassword(16)
		hash, _ := auth.HashPassword(adminPass)
		if err := db.CreateAdmin(adminUser, hash); err != nil {
			log.Fatalf("Failed to create admin: %v", err)
		}
		fmt.Println("╔══════════════════════════════════════════╗")
		fmt.Println("║        NaivePanel — Setup Complete       ║")
		fmt.Println("╠══════════════════════════════════════════╣")
		fmt.Printf("║  Admin Username: %-24s║\n", adminUser)
		fmt.Printf("║  Admin Password: %-24s║\n", adminPass)
		fmt.Printf("║  Panel Port:     %-24d║\n", cfg.PanelPort)
		fmt.Println("╠══════════════════════════════════════════╣")
		fmt.Println("║  ⚠  SAVE THESE CREDENTIALS NOW!  ⚠      ║")
		fmt.Println("╚══════════════════════════════════════════╝")
	} else if *setup {
		fmt.Println("Admin already exists. Setup skipped.")
	}

	if *setup {
		return
	}

	// Initialize default settings if needed
	if _, err := db.GetSetting("port"); err != nil {
		defSettings := models.Settings{
			Domain:    "",
			Port:      443,
			TLSEmail:  "",
			DecoySite: "",
		}
		db.SaveSettings(&defSettings)
	}

	// Initialize NaiveProxy manager
	manager := naiveproxy.NewManager(cfg.NaiveBinary, cfg.CaddyfilePath)

	// Create handlers
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret}
	userHandler := &handlers.UserHandler{DB: db, Manager: manager}
	settingsHandler := &handlers.SettingsHandler{DB: db, Manager: manager}
	statusHandler := &handlers.StatusHandler{DB: db, Manager: manager}

	// Setup routes
	mux := http.NewServeMux()

	// API routes (auth required except login)
	mux.HandleFunc("/api/login", authHandler.Login)
	mux.HandleFunc("/api/users", handlers.AuthMiddleware(cfg.JWTSecret, userHandler.ServeHTTP))
	mux.HandleFunc("/api/users/", handlers.AuthMiddleware(cfg.JWTSecret, userHandler.ServeHTTP))
	mux.HandleFunc("/api/settings", handlers.AuthMiddleware(cfg.JWTSecret, settingsHandler.ServeHTTP))
	mux.HandleFunc("/api/status", handlers.AuthMiddleware(cfg.JWTSecret, statusHandler.GetStatus))
	mux.HandleFunc("/api/service/", handlers.AuthMiddleware(cfg.JWTSecret, statusHandler.ServiceAction))

	// Static files from embedded filesystem
	webContent, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to get web filesystem: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(webContent)))

	// Apply CORS middleware
	handler := handlers.CORSMiddleware(mux)

	// Start server
	addr := fmt.Sprintf("0.0.0.0:%d", cfg.PanelPort)
	log.Printf("NaivePanel starting on http://%s", addr)
	log.Printf("Data directory: %s", cfg.DataDir)

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down...")
		if manager.IsRunning() {
			manager.Stop()
		}
		server.Close()
	}()

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
