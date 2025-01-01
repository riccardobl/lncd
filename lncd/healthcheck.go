package main

import (
	"encoding/json"
	"errors"
	"net/http"
)

type HealthStatus struct {
	Status  string `json:"status"`
	Stats   Stats
	Message string `json:"message"`
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	var stats *Stats = getStats()
	var err error = nil
	if stats == nil {
		err = errors.New("starting")
	}

	if err == nil {
		w.Header().Set("Content-Type", "application/json")
		err = json.NewEncoder(w).Encode(HealthStatus{
			Status:  "OK",
			Stats:   *stats,
			Message: "",
		})
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(HealthStatus{
			Status:  "FAIL",
			Stats:   Stats{},
			Message: err.Error(),
		})

	}
}
