package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"music-server/auth"
	"music-server/database"
	"music-server/scanner"
)

// Global scanner instance
var ScannerInstance *scanner.Scanner

// InitScanner initializes the global scanner instance
func InitScanner(s *scanner.Scanner) {
	ScannerInstance = s
}

// ScanLibrary handles library scanning requests
func ScanLibrary(w http.ResponseWriter, r *http.Request) {
	log.Printf("ScanLibrary called")

	// Check if scanner is initialized
	if ScannerInstance == nil {
		log.Printf("Scanner instance is nil!")
		response := auth.APIResponse{
			Success: false,
			Error:   "Scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Starting scan with scanner instance")

	// Start the scan
	scan, err := ScannerInstance.StartScan()
	if err != nil {
		log.Printf("Error starting scan: %v", err)
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to start scan: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Scan started successfully: %+v", scan)

	response := auth.APIResponse{
		Success: true,
		Message: "Scan started successfully",
		Data:    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetScans handles getting all scans (current and historical)
func GetScans(w http.ResponseWriter, r *http.Request) {
	// Get database instance from scanner
	if ScannerInstance == nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get scan store from scanner
	db := getDBFromScanner()
	if db == nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Database not available",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	scanStore := database.NewScanStore(db)

	// Get all scans from database
	allScans, err := scanStore.GetAllScans()
	if err != nil {
		log.Printf("Error getting all scans: %v", err)
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to get scans: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Separate current scan from completed scans
	var currentScan *database.ScanStatus
	var completedScans []database.ScanStatus

	for _, scan := range allScans {
		if scan.Status == "running" {
			currentScan = &scan
		} else {
			completedScans = append(completedScans, scan)
		}
	}

	// If no current scan in database, check if scanner is currently running
	if currentScan == nil && ScannerInstance.IsScanning() {
		if scanStatus, err := ScannerInstance.GetScanStatus(); err == nil {
			currentScan = scanStatus
		}
	}

	// Return response with both current and completed scans
	responseData := map[string]interface{}{
		"current_scan":    currentScan,
		"completed_scans": completedScans,
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Scans retrieved successfully",
		Data:    responseData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetScanStatus handles getting the current scan status (deprecated, use GetScans)
func GetScanStatus(w http.ResponseWriter, r *http.Request) {
	// Check if scanner is initialized
	if ScannerInstance == nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get scan status
	scan, err := ScannerInstance.GetScanStatus()
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "No scan in progress",
		}
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Scan status retrieved successfully",
		Data:    scan,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StopScan handles stopping the current scan
func StopScan(w http.ResponseWriter, r *http.Request) {
	// Check if scanner is initialized
	if ScannerInstance == nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Stop the scan
	err := ScannerInstance.StopScan()
	if err != nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to stop scan: " + err.Error(),
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	response := auth.APIResponse{
		Success: true,
		Message: "Scan stop requested",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetScanHistory handles getting scan history (deprecated, use GetScans)
func GetScanHistory(w http.ResponseWriter, r *http.Request) {
	// Redirect to GetScans for consistency
	GetScans(w, r)
}

// ClearScans handles clearing all scan history
func ClearScans(w http.ResponseWriter, r *http.Request) {
	// Check if scanner is initialized
	if ScannerInstance == nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Scanner not initialized",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// Get database instance from scanner
	db := getDBFromScanner()
	if db == nil {
		response := auth.APIResponse{
			Success: false,
			Error:   "Database not available",
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	scanStore := database.NewScanStore(db)

	// Clear all scans from database
	err := scanStore.ClearAllScans()
	if err != nil {
		log.Printf("Error clearing scan history: %v", err)
		response := auth.APIResponse{
			Success: false,
			Error:   "Failed to clear scan history: " + err.Error(),
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	log.Printf("Scan history cleared successfully")

	response := auth.APIResponse{
		Success: true,
		Message: "Scan history cleared successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to get database instance from scanner
func getDBFromScanner() *database.DB {
	if ScannerInstance != nil {
		return ScannerInstance.GetDB()
	}
	return nil
}
