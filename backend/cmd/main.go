package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"uber-system/pkg/api"
	"uber-system/pkg/manager"
)

func main() {
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	useRedis := os.Getenv("USE_REDIS") == "true"

	fmt.Println("Uber Geospatial System - Multi-Index Architecture")
	fmt.Printf("Redis enabled: %v\n", useRedis)
	if useRedis {
		fmt.Printf("Redis address: %s\n", redisAddr)
	}

	mgr, err := manager.NewDriverManager(
		18.5204,
		19.0760,
		72.8777,
		72.9982,
		redisAddr,
		useRedis,
	)

	if err != nil {
		log.Fatalf("Failed to initialize manager: %v", err)
	}

	defer mgr.Close()

	handler := api.NewHandler(mgr)

	http.HandleFunc("/drivers", handler.AddDriver)
	http.HandleFunc("/drivers/location", handler.UpdateLocation)
	http.HandleFunc("/drivers/status", handler.UpdateStatus)
	http.HandleFunc("/drivers/search", handler.SearchDrivers)
	http.HandleFunc("/drivers/compare", handler.CompareIndexes)
	http.HandleFunc("/stats", handler.GetStats)
	http.HandleFunc("/health", handler.Health)

	fmt.Println("\nServer starting on :8080")
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  POST   /drivers              - Add new driver")
	fmt.Println("  PUT    /drivers/location     - Update driver location")
	fmt.Println("  PUT    /drivers/status       - Update driver status")
	fmt.Println("  POST   /drivers/search       - Search nearby drivers")
	fmt.Println("  POST   /drivers/compare      - Compare all indexes")
	fmt.Println("  GET    /stats                - Get system statistics")
	fmt.Println("  GET    /health               - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
