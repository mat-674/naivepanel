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
	domain := flag.String("domain", "", "Domain for NaiveProxy (e.g. proxy.example.com)")
	tlsEmail := flag.String("tls-email", "", "Email for Let's Encrypt TLS certificate")
	createUser := flag.Bool("create-user", false, "Create a proxy user during setup")
	proxyUserFlag := flag.String("proxy-user", "", "Proxy user username (auto-generated if empty)")
	proxyPassFlag := flag.String("proxy-pass", "", "Proxy user password (auto-generated if empty)")
	adminUserFlag := flag.String("admin-user", "", "Admin username (auto-generated if empty)")
	adminPassFlag := flag.String("admin-pass", "", "Admin password (auto-generated if empty)")
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
		adminUser := *adminUserFlag
		adminPass := *adminPassFlag
		if adminUser == "" {
			adminUser = config.GenerateRandomUsername(8)
		}
		if adminPass == "" {
			adminPass = config.GenerateRandomPassword(16)
		}
		hash, _ := auth.HashPassword(adminPass)
		if err := db.CreateAdmin(adminUser, hash); err != nil {
			log.Fatalf("Failed to create admin: %v", err)
		}
		fmt.Printf("Admin Username: %s\n", adminUser)
		fmt.Printf("Admin Password: %s\n", adminPass)
		fmt.Printf("Panel Port: %d\n", cfg.PanelPort)
	} else if *setup {
		fmt.Println("Admin already exists. Setup skipped.")
	}

	// Save domain/TLS settings if provided
	if *domain != "" || *tlsEmail != "" {
		settings := models.Settings{
			Domain:   *domain,
			Port:     443,
			TLSEmail: *tlsEmail,
		}
		if err := db.SaveSettings(&settings); err != nil {
			log.Printf("Failed to save settings: %v", err)
		} else {
			fmt.Printf("Domain: %s\n", *domain)
			fmt.Printf("TLS Email: %s\n", *tlsEmail)
		}
	}

	// Create a proxy user if requested
	if *createUser {
		proxyUser := *proxyUserFlag
		proxyPass := *proxyPassFlag
		if proxyUser == "" {
			proxyUser = config.GenerateRandomUsername(8)
		}
		if proxyPass == "" {
			proxyPass = config.GenerateRandomPassword(16)
		}
		if _, err := db.CreateUser(models.CreateUserRequest{Username: proxyUser, Password: proxyPass}); err != nil {
			log.Printf("Failed to create proxy user: %v", err)
		} else {
			fmt.Printf("Proxy Username: %s\n", proxyUser)
			fmt.Printf("Proxy Password: %s\n", proxyPass)
		}
	}

	// Generate initial Caddyfile if domain was provided during setup
	if *setup && *domain != "" {
		settings, _ := db.GetSettings()
		users, _ := db.GetEnabledUsers()
		if settings != nil {
			content, err := naiveproxy.GenerateCaddyfile(settings, users)
			if err == nil {
				manager := naiveproxy.NewManager(cfg.NaiveBinary, cfg.CaddyfilePath)
				if err := manager.WriteCaddyfile(content); err != nil {
					log.Printf("Failed to write initial Caddyfile: %v", err)
				} else {
					fmt.Println("Initial Caddyfile generated.")
				}
			}
		}
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
	subHandler := &handlers.SubHandler{DB: db}

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
	fileServer := http.FileServer(http.FS(webContent))

	// Dynamic interceptor for root
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		settings, err := db.GetSettings()
		subPathPrefix := "/sub/" // fallback default
		if err == nil && settings != nil && settings.SubPath != "" {
			subPathPrefix = "/" + settings.SubPath + "/"
		}

		if strings.HasPrefix(r.URL.Path, subPathPrefix) {
			subHandler.ServeHTTP(w, r)
			return
		}
		
		// Fallback to static files
		fileServer.ServeHTTP(w, r)
	})

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
