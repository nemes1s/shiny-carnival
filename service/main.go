package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Record struct {
	ID            string `json:"id"`
	AssetName     string `json:"asset_name"`
	IP            string `json:"ip"`
	CreatedUTC    string `json:"created_utc"`
	Source        string `json:"source"`
	Category      string `json:"category"`
	ASN           string `json:"asn"`
	CorrelationID int    `json:"correlationId"`
}

var recordChannel = make(chan Record, 20)

func main() {
	http.HandleFunc("/process", processHandler)
	go startSendingToAnalytics()

	log.Println("Microservice running on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func processHandler(w http.ResponseWriter, r *http.Request) {

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		log.Printf("Error reading request body: %v", err)
		return
	}
	defer r.Body.Close()

	var record Record
	if err := json.Unmarshal(body, &record); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		log.Printf("Error unmarshalling JSON: %v", err)
		return
	}

	enrichedRecord, err := enrichRecord(record)
	if err != nil {
		http.Error(w, "Failed to enrich record", http.StatusInternalServerError)
		log.Printf("Error enriching record with ID %s: %v", record.ID, err)
		return
	}

	recordChannel <- enrichedRecord
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"status": "record processed"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func enrichRecord(record Record) (Record, error) {
	url := "https://api.heyering.com/enrichment"
	requestBody, err := json.Marshal(map[string]interface{}{
		"id":       record.ID,
		"asset":    record.AssetName,
		"ip":       record.IP,
		"category": record.Category,
	})
	if err != nil {
		return record, err
	}

	maxRetries := 3
	retryDelay := time.Second
	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			return record, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "eye-am-hiring")

		log.Printf("Attempt %d: Calling Enrichment Service for record ID %s", attempt, record.ID)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Attempt %d: Error calling Enrichment Service: %v", attempt, err)
		} else {
			defer resp.Body.Close()
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("Attempt %d: Error reading response body: %v", attempt, err)
			} else if resp.StatusCode == http.StatusOK {
				var enrichedData struct {
					ASN           string `json:"asn"`
					Category      string `json:"category"`
					CorrelationID int    `json:"correlationId"`
				}
				if err := json.Unmarshal(responseBody, &enrichedData); err != nil {
					log.Printf("Attempt %d: Error unmarshalling response: %v", attempt, err)
				} else {
					// Success
					record.ASN = enrichedData.ASN
					record.Category = enrichedData.Category
					record.CorrelationID = enrichedData.CorrelationID
					return record, nil
				}
			} else {
				log.Printf("Attempt %d: Enrichment service returned status %d, response: %s", attempt, resp.StatusCode, responseBody)
			}
		}

		if attempt < maxRetries {
			time.Sleep(retryDelay)
			retryDelay *= 2
		} else {
			return record, fmt.Errorf("failed to enrich record after %d attempts", maxRetries)
		}
	}
	return record, fmt.Errorf("unexpected error during enrichment")
}

func startSendingToAnalytics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var batch []Record

	for {
		select {
		case record := <-recordChannel:
			batch = append(batch, record)
		case <-ticker.C:
			if len(batch) > 0 {
				var recordsToSend []Record
				if len(batch) >= 20 {
					recordsToSend = batch[:20]
					batch = batch[20:]
				} else {
					recordsToSend = batch
					batch = batch[:0]
				}
				if err := sendToAnalytics(recordsToSend); err != nil {
					log.Printf("Failed to send batch to Analytics Service: %v", err)
				}
			}
		}
	}
}

func sendToAnalytics(records []Record) error {
	url := "https://api.heyering.com/analytics"
	requestBody, err := json.Marshal(records)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "eye-am-hiring")

	log.Printf("Sending batch of %d records to Analytics Service", len(records))
	log.Printf("Request Payload: %s", requestBody)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("analytics service returned status %d, response: %s", resp.StatusCode, responseBody)
	}

	var response struct {
		Status        string `json:"status"`
		ItemsIngested int    `json:"itemsIngested"`
	}
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return err
	}

	log.Printf("Successfully sent batch to Analytics Service. Items Ingested: %d", response.ItemsIngested)
	return nil
}
