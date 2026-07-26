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
	"strings"
	"syscall"
	"time"
)

//go:embed web/*
var webFS embed.FS

func main() {
	configPath := flag.String("config", "", "Path to config file")
	dataDir := flag.String("data-dir", "", "Data directory")
	port := flag.Int("port", 0, "Panel port (overrides config)")
	bind := flag.String("bind", "", "Loopback address for the panel (overrides config)")
	setup := flag.Bool("setup", false, "Initialize database and print credentials, then exit")
	startProxy := flag.Bool("start-proxy", false, "Start NaiveProxy as a child process at boot (for non-systemd hosts such as containers)")
	domain := flag.String("domain", "", "Domain for NaiveProxy (e.g. proxy.example.com)")
	tlsEmail := flag.String("tls-email", "", "Email for Let's Encrypt TLS certificate")
	createUser := flag.Bool("create-user", false, "Create a proxy user during setup")
	proxyUserFlag := flag.String("proxy-user", "", "Proxy user username (auto-generated if empty)")
	proxyPassFlag := flag.String("proxy-pass", "", "Proxy user password (auto-generated if empty)")
	adminUserFlag := flag.String("admin-user", "", "Admin username (auto-generated if empty)")
	adminPassFlag := flag.String("admin-pass", "", "Admin password (auto-generated if empty)")
	panelPublicEnable := flag.Bool("panel-public-enable", false, "Publish the panel at /<path>/* behind HTTP Basic Auth, then exit")
	panelPublicDisable := flag.Bool("panel-public-disable", false, "Unpublish the panel (back to loopback only), then exit")
	panelPublicPathFlag := flag.String("panel-public-path", "", "Path prefix for --panel-public-enable (default: panel)")
	panelBasicUser := flag.String("panel-basic-user", "", "Basic Auth username for the published panel (auto-generated if empty)")
	panelBasicPass := flag.String("panel-basic-pass", "", "Basic Auth password for the published panel (auto-generated if empty)")
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
		cfg.OverrideDataDir(*dataDir)
		if err := cfg.Save(cfgPath); err != nil {
			log.Fatalf("Failed to save configuration after data directory override: %v", err)
		}
	}
	if *port != 0 {
		cfg.PanelPort = *port
	}
	if *bind != "" {
		cfg.OverridePanelBind(*bind)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
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
		normalizedDomain, err := naiveproxy.NormalizeDomain(*domain)
		if err != nil {
			log.Fatalf("Invalid domain: %v", err)
		}
		normalizedTLSEmail, err := naiveproxy.NormalizeTLSEmail(*tlsEmail)
		if err != nil {
			log.Fatalf("Invalid TLS email: %v", err)
		}
		settings := models.Settings{
			Domain:   normalizedDomain,
			Port:     443,
			TLSEmail: normalizedTLSEmail,
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
		if err := naiveproxy.ValidateProxyUsername(proxyUser); err != nil {
			log.Fatalf("Invalid proxy username: %v", err)
		}
		if err := naiveproxy.ValidateProxyPassword(proxyPass); err != nil {
			log.Fatalf("Invalid proxy password: %v", err)
		}
		if _, err := db.CreateUser(models.CreateUserRequest{Username: proxyUser, Password: proxyPass}); err != nil {
			log.Printf("Failed to create proxy user: %v", err)
		} else {
			fmt.Printf("Proxy Username: %s\n", proxyUser)
			fmt.Printf("Proxy Password: %s\n", proxyPass)
		}
	}

	// Generate an initial Caddyfile during setup so the dedicated NaiveProxy
	// service has a valid configuration even before the first settings update.
	if *setup {
		settings, _ := db.GetSettings()
		users, _ := db.GetEnabledUsers()
		if settings != nil {
			content, err := naiveproxy.GenerateCaddyfile(settings, users, cfg.PanelUpstream())
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

	// --- Panel publication toggles. Each writes settings, regenerates the
	// Caddyfile (so Caddy picks up the new route immediately), and exits so
	// these commands can be scripted. ---
	if *panelPublicEnable {
		if err := enablePanelPublic(db, cfg, *panelPublicPathFlag, *panelBasicUser, *panelBasicPass); err != nil {
			log.Fatalf("Failed to publish panel: %v", err)
		}
		return
	}
	if *panelPublicDisable {
		if err := disablePanelPublic(db, cfg); err != nil {
			log.Fatalf("Failed to unpublish panel: %v", err)
		}
		return
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

	// On non-systemd hosts (e.g. containers) the panel supervises NaiveProxy
	// itself. Under systemd the dedicated unit owns the lifecycle, so the flag
	// is ignored to avoid a second, unmanaged process.
	if *startProxy {
		if manager.UsesSystemd() {
			log.Println("--start-proxy ignored: NaiveProxy is managed by systemd")
		} else {
			startProxyProcess(db, manager, cfg.PanelUpstream())
		}
	}

	// Create handlers
	authHandler := &handlers.AuthHandler{DB: db, JWTSecret: cfg.JWTSecret}
	userHandler := &handlers.UserHandler{DB: db, Manager: manager, PanelUpstream: cfg.PanelUpstream()}
	settingsHandler := &handlers.SettingsHandler{DB: db, Manager: manager, PanelUpstream: cfg.PanelUpstream()}
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

	// Start server
	addr := cfg.PanelAddress()
	log.Printf("NaivePanel starting on http://%s", addr)
	log.Printf("Data directory: %s", cfg.DataDir)

	server := &http.Server{
		Addr:              addr,
		Handler:           handlers.Harden(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
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

// startProxyProcess writes a fresh Caddyfile from the current settings and
// starts NaiveProxy as a child process. Failures are logged, not fatal: the
// panel must stay up so the operator can fix the configuration from the UI.
func startProxyProcess(db *database.DB, manager *naiveproxy.Manager, panelUpstream string) {
	settings, err := db.GetSettings()
	if err != nil {
		log.Printf("Autostart skipped: failed to read settings: %v", err)
		return
	}
	users, err := db.GetEnabledUsers()
	if err != nil {
		log.Printf("Autostart skipped: failed to read users: %v", err)
		return
	}

	content, err := naiveproxy.GenerateCaddyfile(settings, users, panelUpstream)
	if err != nil {
		log.Printf("Autostart skipped: failed to generate Caddyfile: %v", err)
		return
	}
	if err := manager.WriteCaddyfile(content); err != nil {
		log.Printf("Autostart skipped: failed to write Caddyfile: %v", err)
		return
	}

	if err := manager.Start(); err != nil {
		log.Printf("Autostart: failed to start NaiveProxy: %v", err)
		return
	}
	log.Println("NaiveProxy started")
}

// regenerateCaddyfileAndReload is a helper used by enablePanelPublic and
// disablePanelPublic to regenerate the Caddyfile after settings changes.
func regenerateCaddyfileAndReload(db *database.DB, manager *naiveproxy.Manager, panelUpstream string) error {
	settings, err := db.GetSettings()
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}
	users, err := db.GetEnabledUsers()
	if err != nil {
		return fmt.Errorf("read users: %w", err)
	}
	content, err := naiveproxy.GenerateCaddyfile(settings, users, panelUpstream)
	if err != nil {
		return fmt.Errorf("generate caddyfile: %w", err)
	}
	if err := manager.WriteCaddyfile(content); err != nil {
		return fmt.Errorf("write caddyfile: %w", err)
	}
	if err := manager.Reload(); err != nil {
		log.Printf("NaiveProxy is not running, so the change will activate on its next start: %v", err)
	}
	return nil
}

// enablePanelPublic turns on the public /<path>/* route and writes a fresh
// Caddyfile so Caddy picks it up on its next reload. Auto-generates Basic
// Auth credentials when the corresponding flags are empty.
func enablePanelPublic(db *database.DB, cfg *config.Config, path, user, pass string) error {
	settings, err := db.GetSettings()
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}
	if settings.Domain == "" {
		return fmt.Errorf("a Domain must be configured before publishing the panel (run the installer or set Domain via API)")
	}

	if path == "" {
		path = "panel"
	}
	path = strings.Trim(path, "/")
	if path == "" || strings.Contains(path, "/") {
		return fmt.Errorf("panel publish path must be a single URL-safe segment (got %q)", path)
	}
	if user == "" {
		user = config.GenerateRandomUsername(8)
	}
	if pass == "" {
		pass = config.GenerateRandomPassword(20)
	}
	if err := naiveproxy.ValidateProxyUsername(user); err != nil {
		return fmt.Errorf("panel basic auth user: %w", err)
	}
	if err := naiveproxy.ValidateProxyPassword(pass); err != nil {
		return fmt.Errorf("panel basic auth password: %w", err)
	}

	hash, err := auth.HashPassword(pass)
	if err != nil {
		return fmt.Errorf("hash basic auth password: %w", err)
	}

	settings.PanelPublic = true
	settings.PanelPublicPath = path
	settings.PanelBasicUser = user
	settings.PanelBasicHash = hash
	if err := db.SaveSettings(settings); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}

	manager := naiveproxy.NewManager(cfg.NaiveBinary, cfg.CaddyfilePath)
	if err := regenerateCaddyfileAndReload(db, manager, cfg.PanelUpstream()); err != nil {
		return err
	}

	scheme := "https"
	if settings.TLSEmail == "" {
		scheme = "https" // Caddy's `tls internal` still serves HTTPS with a self-signed cert
	}
	url := fmt.Sprintf("%s://%s/%s/", scheme, settings.Domain, path)
	fmt.Println("Panel published.")
	fmt.Printf("Panel Public URL: %s\n", url)
	fmt.Printf("Basic Auth User:  %s\n", user)
	fmt.Printf("Basic Auth Pass:  %s\n", pass)
	fmt.Println("If NaiveProxy was already running it has been reloaded; otherwise the route activates on next start.")
	return nil
}

// disablePanelPublic flips the flag off, regenerates the Caddyfile without
// the route block, and exits.
func disablePanelPublic(db *database.DB, cfg *config.Config) error {
	settings, err := db.GetSettings()
	if err != nil {
		return fmt.Errorf("read settings: %w", err)
	}
	if !settings.PanelPublic {
		fmt.Println("Panel is already loopback-only; nothing to do.")
		return nil
	}

	settings.PanelPublic = false
	settings.PanelPublicPath = ""
	settings.PanelBasicUser = ""
	settings.PanelBasicHash = ""
	if err := db.SaveSettings(settings); err != nil {
		return fmt.Errorf("save settings: %w", err)
	}

	manager := naiveproxy.NewManager(cfg.NaiveBinary, cfg.CaddyfilePath)
	if err := regenerateCaddyfileAndReload(db, manager, cfg.PanelUpstream()); err != nil {
		return err
	}

	fmt.Println("Panel unpublished. It is reachable only via SSH tunnel on 127.0.0.1.")
	fmt.Println("If NaiveProxy was already running it has been reloaded; otherwise the change activates on next start.")
	return nil
}
