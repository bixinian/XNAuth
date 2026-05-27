package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"xnauth/internal/admin"
	"xnauth/internal/config"
	"xnauth/internal/database"
	logx "xnauth/internal/log"
	"xnauth/internal/router"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config yaml")
	initAdmin := flag.Bool("init-admin", false, "create or update an admin user then exit")
	adminUsername := flag.String("admin-username", "admin", "admin username for -init-admin")
	adminPassword := flag.String("admin-password", "", "admin password for -init-admin")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	logger, cleanup, err := logx.New(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger failed: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	db, err := database.ConnectMySQL(cfg.MySQL, logger)
	if err != nil {
		logger.Fatal("connect mysql failed", zap.Error(err))
	}

	if *initAdmin {
		if err := admin.EnsureAdminUser(db, *adminUsername, *adminPassword); err != nil {
			logger.Fatal("init admin failed", zap.Error(err))
		}
		logger.Info("admin user initialized", zap.String("username", *adminUsername))
		return
	}

	engine := router.New(router.Dependencies{
		Config: cfg,
		Logger: logger,
		DB:     db,
	})

	server := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("server starting", zap.String("addr", cfg.Server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server stopped unexpectedly", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logger.Info("server shutting down")
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server shutdown failed", zap.Error(err))
	}
}
