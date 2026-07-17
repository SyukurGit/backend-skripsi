package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"support-backend/config"
	"support-backend/database"
	"support-backend/internal/delivery/http"
	wsdelivery "support-backend/internal/delivery/websocket"
	"support-backend/internal/middleware"
	"support-backend/internal/repository/mysql"
	"support-backend/internal/usecase"
)

func main() {
	cfg := config.Load()

	db, err := database.ConnectAndBootstrapMySQL(cfg)
	if err != nil {
		log.Fatalf("gagal konek database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("gagal migrate: %v", err)
	}

	if err := database.SeedIfEmpty(db); err != nil {
		log.Fatalf("gagal seed: %v", err)
	}

	userRepo := mysql.NewUserRepository(db)
	profileRepo := mysql.NewUserProfileRepository(db)
	ticketRepo := mysql.NewTicketRepository(db)
	messageRepo := mysql.NewMessageRepository(db)
	jitRepo := mysql.NewJITSessionRepository(db)
	auditRepo := mysql.NewAuditLogRepository(db)

	auditHub := wsdelivery.NewAuditHub()
	chatHub := wsdelivery.NewChatHub()
	terminalHub := wsdelivery.NewTerminalHub()
	go auditHub.Run()
	go chatHub.Run()
	go terminalHub.Run()

	auditPublisher := wsdelivery.NewAuditPublisher(auditHub)
	chatPublisher := wsdelivery.NewChatPublisher(chatHub)
	terminalPublisher := wsdelivery.NewTerminalPublisher(terminalHub)

	auditUC := usecase.NewAuditUsecase(auditRepo, auditPublisher)
	authUC := usecase.NewAuthUsecase(userRepo, profileRepo, cfg)
	ticketUC := usecase.NewTicketUsecase(ticketRepo, jitRepo, auditUC, terminalPublisher)
	messageUC := usecase.NewMessageUsecase(ticketRepo, messageRepo, auditUC, chatPublisher, terminalPublisher)
	jitUC := usecase.NewJITUsecase(ticketRepo, jitRepo, auditUC, messageRepo, chatPublisher, terminalPublisher)
	csUC := usecase.NewCSUsecase(userRepo, profileRepo, ticketRepo, messageRepo, chatPublisher, jitUC, auditUC, terminalPublisher)
	adminUC := usecase.NewAdminUsecase(userRepo, profileRepo, ticketRepo, auditRepo, terminalPublisher)

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORS(cfg))
	// Keamanan: jangan percaya semua proxy secara default.
	_ = r.SetTrustedProxies(nil)

	http.RegisterRoutes(r, cfg, authUC, ticketUC, messageUC, jitUC, auditUC, csUC, adminUC, ticketRepo, chatHub, auditHub, terminalHub)

	addr := ":" + cfg.AppPort
	log.Printf("server listen %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server gagal start: %v", err)
	}
}








