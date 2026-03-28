package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ncvms/internal/config"
	"ncvms/internal/db"
	"ncvms/internal/handlers"
	"ncvms/internal/messaging"
	"ncvms/internal/middleware"
	"ncvms/internal/router"
	"ncvms/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("config:", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal("db:", err)
	}
	defer pool.Close()

	usersStore := store.NewUserStore(pool)
	childStore := store.NewChildStore(pool)
	childLinkOTPStore := store.NewChildLinkOTPStore(pool)
	userMobileChangeOTPStore := store.NewUserMobileChangeOTPStore(pool)

	// OTP sender now uses TextLK SMS API.
	whatsAppSender := messaging.NewTextLKSender()

	authHandler := &handlers.AuthHandler{
		UserStore:  usersStore,
		AuditStore: store.NewAuditStore(pool),
		JWTSecret:  cfg.JWTSecret,
		JWTExpiry:  cfg.JWTExpiryHours,
	}
	usersHandler := &handlers.UsersHandler{
		UserStore:             usersStore,
		UserMobileChangeStore: userMobileChangeOTPStore,
		WhatsAppSender:        whatsAppSender,
		PHMLoginURL:           cfg.PHMLoginURL,
		OTPTTL:                time.Duration(cfg.MobileChangeOTPTTLMin) * time.Minute,
		OTPResendCooldown:     time.Duration(cfg.MobileChangeOTPCooldownSec) * time.Second,
		OTPMaxAttempts:        cfg.MobileChangeOTPMaxAttempts,
	}

	childrenHandler := &handlers.ChildrenHandler{
		ChildStore:        childStore,
		UserStore:         usersStore,
		ChildLinkOTPStore: childLinkOTPStore,
		WhatsAppSender:    whatsAppSender,
		OTPTTL:            time.Duration(cfg.ChildLinkOTPTTLMin) * time.Minute,
		OTPResendCooldown: time.Duration(cfg.ChildLinkOTPCooldownSec) * time.Second,
		OTPMaxAttempts:    cfg.ChildLinkOTPMaxAttempts,
		ParentPortalLink:  cfg.ParentPortalLink,
	}
	vaccinesHandler := &handlers.VaccinesHandler{VaccineStore: store.NewVaccineStore(pool)}
	vaccRecHandler := &handlers.VaccinationRecordsHandler{RecordStore: store.NewVaccinationRecordStore(pool)}
	schedHandler := &handlers.SchedulesHandler{ScheduleStore: store.NewScheduleStore(pool)}
	growthHandler := &handlers.GrowthHandler{GrowthStore: store.NewGrowthRecordStore(pool)}
	notifHandler := &handlers.NotificationsHandler{NotificationStore: store.NewNotificationStore(pool)}
	reportsHandler := &handlers.ReportsHandler{ReportStore: store.NewReportStore(pool)}
	auditHandler := &handlers.AuditHandler{AuditStore: store.NewAuditStore(pool)}
	analyticsHandler := &handlers.AnalyticsHandler{
		ChildStore:  childStore,
		RecordStore: store.NewVaccinationRecordStore(pool),
		GrowthStore: store.NewGrowthRecordStore(pool),
		NotifyStore: store.NewNotificationStore(pool),
	}

	engine := gin.New()
	engine.Use(gin.Logger(), middleware.Recovery())
	router.Setup(engine, cfg.JWTSecret, authHandler, usersHandler, childrenHandler, vaccinesHandler,
		vaccRecHandler, schedHandler, growthHandler, notifHandler, reportsHandler, auditHandler, analyticsHandler)

	srv := &http.Server{Addr: ":" + cfg.Port, Handler: engine}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server:", err)
		}
	}()
	fmt.Println("NCVMS API listening on :" + cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	if err := srv.Shutdown(ctx); err != nil {
		log.Println("shutdown:", err)
	}
}
