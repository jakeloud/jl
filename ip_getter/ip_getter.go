package ip_getter

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
)

// GetPublicIP retrieves the public IP address from ip4.me/api
func GetPublicIP() (string, error) {
	resp, err := http.Get("http://ip4.me/api/")
	if err != nil {
		return "", fmt.Errorf("failed to fetch IP: %v", err)
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	record, err := reader.Read()
	if err != nil {
		return "", fmt.Errorf("failed to read CSV: %v", err)
	}

	if len(record) < 2 {
		return "", fmt.Errorf("invalid CSV format: expected at least 2 columns")
	}

	return record[1], nil
}
