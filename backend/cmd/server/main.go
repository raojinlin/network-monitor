package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/raojinlin/traffic-sniff/internal/capture"
	"github.com/raojinlin/traffic-sniff/internal/handlers"
	"github.com/raojinlin/traffic-sniff/internal/middleware"
	"github.com/raojinlin/traffic-sniff/internal/storage"
)

var (
	port      = flag.String("port", "8080", "Server port")
	iface     = flag.String("interface", "", "Network interface to capture (empty for all)")
	storePath = flag.String("storage", "./data", "Path to store historical data")
)

func main() {
	flag.Parse()

	log.Printf("Traffic Monitor Server starting...")
	log.Printf("Configuration: port=%s, interface=%s, storage=%s", *port, *iface, *storePath)

	// Initialize storage
	log.Printf("Initializing storage systems...")
	store := storage.NewMemoryStorage()
	historicalStore := storage.NewFileStorage(*storePath)
	log.Printf("Storage systems initialized")

	// Initialize capture manager
	log.Printf("Initializing packet capture manager...")
	captureManager := capture.NewManager(store)
	
	// Start capture on the specified interface
	log.Printf("Starting packet capture on interface '%s'...", *iface)
	if err := captureManager.Start(*iface); err != nil {
		log.Fatalf("Failed to start packet capture: %v", err)
	}
	defer captureManager.Stop()
	log.Printf("Packet capture started successfully")

	// Initialize HTTP handlers
	log.Printf("Setting up HTTP handlers...")
	handler := handlers.NewHandler(store, historicalStore, captureManager)
	
	// Setup routes
	log.Printf("Setting up API routes...")
	mux := http.NewServeMux()
	mux.HandleFunc("/api/traffic/realtime", handler.RealtimeTraffic)
	mux.HandleFunc("/api/traffic/connections", handler.ConnectionList)
	mux.HandleFunc("/api/traffic/history", handler.HistoricalTraffic)
	mux.HandleFunc("/api/interfaces", handler.ListInterfaces)
	mux.HandleFunc("/api/interfaces/switch", handler.SwitchInterface)
	mux.HandleFunc("/ws", handler.WebSocketHandler)
	
	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir("../frontend/dist")))
	log.Printf("API routes configured")

	// Apply logging middleware
	log.Printf("Applying logging middleware...")
	loggedMux := middleware.LoggingMiddleware(mux)
	wsLoggedMux := middleware.WebSocketLoggingMiddleware(loggedMux)
	
	server := &http.Server{
		Addr:    ":" + *port,
		Handler: wsLoggedMux,
	}

	// Start server
	go func() {
		log.Printf("Server starting on port %s", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	// Graceful shutdown
	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}