package main

import (
	"fmt"
	"log"
	"net/http"
	basic "uber-system/phase1-geospatial/1.1-basic-indexing"
)

func main() {
	manager := basic.NewDriverManager(
		40.4774,
		40.9176,
		-74.2591,
		-73.7004,
	)

	handler := basic.NewAPIHandler(manager)

	http.HandleFunc("/drivers", handler.AddDriverHandler)
	http.HandleFunc("/drivers/location", handler.UpdateLocationHandler)
	http.HandleFunc("/drivers/search", handler.SearchDriversHandler)
	http.HandleFunc("/drivers/status", handler.UpdateStatusHandler)
	http.HandleFunc("/stats", handler.StatsHandler)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	fmt.Println("Uber Geospatial System - Phase 1.1")
	fmt.Println("========================================")
	fmt.Println("Server starting on :8080")
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  POST   /drivers              - Add new driver")
	fmt.Println("  PUT    /drivers/location     - Update driver location")
	fmt.Println("  POST   /drivers/search       - Search nearby drivers")
	fmt.Println("  PUT    /drivers/status       - Update driver status")
	fmt.Println("  GET    /stats                - Get system statistics")
	fmt.Println("  GET    /health               - Health check")
	fmt.Println("\nPress Ctrl+C to stop")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

