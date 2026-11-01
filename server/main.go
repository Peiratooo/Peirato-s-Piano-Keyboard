package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Server struct {
	db            *gorm.DB
	hub           *Hub
	dataDir       string
	origins       map[string]bool
	limiter       *IPLimiter
	desktopMu     sync.Mutex
	desktopGrants map[string]*DesktopGrant
}

func config(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
func setupDatabase() (*gorm.DB, error) {
	endpoint := &url.URL{Scheme: "postgres", Host: config("DB_HOST", "127.0.0.1") + ":" + config("DB_PORT", "5432"), Path: "/" + config("DB_NAME", "piano"), User: url.UserPassword(config("DB_USER", "postgres"), os.Getenv("DB_PASSWORD"))}
	query := endpoint.Query()
	query.Set("sslmode", config("DB_SSLMODE", "disable"))
	query.Set("TimeZone", config("DB_TIMEZONE", "Asia/Shanghai"))
	query.Set("connect_timeout", "8")
	endpoint.RawQuery = query.Encode()
	cfg, err := pgx.ParseConfig(endpoint.String())
	if err != nil {
		return nil, err
	}
	return gorm.Open(postgres.New(postgres.Config{Conn: stdlib.OpenDB(*cfg)}), &gorm.Config{})
}
func main() {
	inspect := flag.Bool("inspect-db", false, "read database metadata without creating tables")
	flag.Parse()
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		log.Fatal("environment configuration unavailable")
	}

	if err := configureTrustedProxies(config("TRUSTED_PROXY_CIDRS", "")); err != nil {
		log.Fatal("reverse proxy configuration: ", err)
	}
	db, err := setupDatabase()
	if err != nil {
		log.Fatal("database: ", err)
	}
	if *inspect {
		var rows []map[string]any
		if err := db.Raw("SELECT current_database() AS database, version(), has_schema_privilege(current_user, 'public', 'CREATE') AS can_create").Scan(&rows).Error; err != nil {
			log.Fatal(err)
		}
		log.Printf("Database: %v", rows)
		rows = nil
		if err := db.Raw("SELECT table_schema, table_name FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema') ORDER BY 1,2").Scan(&rows).Error; err != nil {
			log.Fatal(err)
		}
		log.Printf("Existing tables: %v", rows)
		return
	}
	if err := db.AutoMigrate(&User{}, &Session{}, &Room{}, &MIDIFile{}, &PracticeLog{}, &Report{}, &MIDIInteraction{}); err != nil {
		log.Fatal("initialize schema: ", err)
	}
	dir := config("DATA_DIR", "./data")
	if absolute, err := filepath.Abs(dir); err == nil {
		dir = absolute
	}
	if err := os.MkdirAll(filepath.Join(dir, "midi"), 0700); err != nil {
		log.Fatal(err)
	}
	s := &Server{db: db, dataDir: dir, origins: map[string]bool{}, limiter: NewIPLimiter()}
	for _, origin := range strings.Split(config("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173"), ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			s.origins[origin] = true
		}
	}
	if err := s.bootstrapAdmin(); err != nil {
		log.Fatal("admin bootstrap: ", err)
	}
	s.desktopGrants = make(map[string]*DesktopGrant)
	s.hub = NewHub(s)
	go s.hub.runClock()
	mux := http.NewServeMux()
	s.routes(mux)
	srv := &http.Server{Addr: config("ADDR", ":8080"), Handler: s.cors(mux), ReadHeaderTimeout: 8 * time.Second, IdleTimeout: 75 * time.Second, MaxHeaderBytes: 1 << 20}
	log.Printf("PP Piano API listening on %s, CORS origins: %d", srv.Addr, len(s.origins))
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "time": time.Now().UTC()})
	})
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.HandleFunc("PATCH /api/auth/me", s.updateProfile)
	mux.HandleFunc("POST /api/auth/avatar", s.uploadAvatar)
	mux.HandleFunc("GET /api/avatars/{name}", s.getAvatar)
	mux.HandleFunc("POST /api/auth/desktop/start", s.startDesktopAuth)
	mux.HandleFunc("POST /api/auth/desktop/confirm", s.confirmDesktopAuth)
	mux.HandleFunc("POST /api/auth/desktop/poll", s.pollDesktopAuth)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/rooms", s.listRooms)
	mux.HandleFunc("GET /api/rooms/random", s.randomRoom)
	mux.HandleFunc("POST /api/rooms", s.createRoom)
	mux.HandleFunc("PATCH /api/rooms/{id}", s.updateRoom)
	mux.HandleFunc("DELETE /api/rooms/{id}", s.closeRoom)
	mux.HandleFunc("GET /api/midis", s.listMIDIs)
	mux.HandleFunc("PUT /api/midis/{id}/interaction", s.setMIDIInteraction)
	mux.HandleFunc("POST /api/midis/{id}/events", s.recordMIDIEvent)
	mux.HandleFunc("POST /api/midis", s.uploadMIDI)
	mux.HandleFunc("GET /api/midis/{id}", s.getMIDI)
	mux.HandleFunc("GET /api/midis/{id}/file", s.downloadMIDI)
	mux.HandleFunc("PATCH /api/midis/{id}", s.updateMIDI)
	mux.HandleFunc("DELETE /api/midis/{id}", s.deleteMIDI)
	mux.HandleFunc("GET /api/admin/midis", s.pendingMIDIs)
	mux.HandleFunc("POST /api/admin/midis/{id}/review", s.reviewMIDI)
	mux.HandleFunc("POST /api/practice/start", s.startPractice)
	mux.HandleFunc("POST /api/practice/heartbeat", s.heartbeatPractice)
	mux.HandleFunc("POST /api/practice/finish", s.finishPractice)
	mux.HandleFunc("GET /api/practice/history", s.practiceHistory)
	mux.HandleFunc("GET /api/practice/stats", s.practiceStats)
	mux.HandleFunc("POST /api/reports", s.createReport)
	mux.HandleFunc("GET /api/admin/reports", s.listReports)
	mux.HandleFunc("POST /api/admin/reports/{id}/resolve", s.resolveReport)
	mux.HandleFunc("GET /ws", s.serveWS)
}
func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && s.origins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		}
		if r.Method == http.MethodOptions {
			if origin != "" && !s.origins[origin] {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			w.WriteHeader(204)
			return
		}
		if origin != "" && !s.origins[origin] {
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
