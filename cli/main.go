package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

type Record struct {
	ID         string
	AssetName  string
	IP         string
	CreatedUTC string
	Source     string
	Category   string
}

var categoryMap = map[string]string{
	"contentinjection":  "contentinjection",
	"content injection": "contentinjection",
	"content_injection": "contentinjection",

	"drivebycompromise":    "drivebycompromise",
	"drive by compromise":  "drivebycompromise",
	"drive-by-compromise":  "drivebycompromise",
	"compromise (driveby)": "drivebycompromise",

	"exploitpublicfacingapplication": "exploitpublicfacingapplication",
	"exploit public facing":          "exploitpublicfacingapplication",
	"explaoit-public facing":         "exploitpublicfacingapplication", // Typo corrected

	"externalremoteservices":  "externalremoteservices",
	"external remote service": "externalremoteservices",
	"external-remote-service": "externalremoteservices",

	"phishing": "phishing",
	"phising":  "phishing",
	"Phising":  "phishing",

	"replicationthroughremovablemedia":    "replicationthroughremovablemedia",
	"replication through removable media": "replicationthroughremovablemedia",
	"Replication through removable media": "replicationthroughremovablemedia",
	"replication-through-removable-media": "replicationthroughremovablemedia",

	"supplychaincompromise":   "supplychaincompromise",
	"supply chain compromise": "supplychaincompromise",
	"supply_chain_compromise": "supplychaincompromise",

	"trustedrelationship":  "trustedrelationship",
	"trusted relationship": "trustedrelationship",
	"trusted-relationship": "trustedrelationship",

	"validaccounts":   "validaccounts",
	"valid accounts":  "validaccounts",
	"valid-accounts":  "validaccounts",
	"valida_accounts": "validaccounts",
}

func sanitizeCategory(category string) string {
	normalizedCategory := strings.TrimSpace(strings.ToLower(category))
	if correctCategory, exists := categoryMap[normalizedCategory]; exists {
		return correctCategory
	}

	return normalizedCategory
}

func main() {
	csvFile := flag.String("file", "data.csv", "Path to the CSV file")
	filterCategory := flag.String("category", "", "Filter by category (optional)")
	filterId := flag.String("id", "", "Filter by id (optional)")

	flag.Parse()

	records, skippedRecords, err := readCSV(*csvFile)
	if err != nil {
		log.Fatalf("Error reading CSV: %v", err)
	}

	// Filter records if a filter is provided
	if *filterCategory != "" {
		records = filterRecordsByCategory(records, *filterCategory)
	}

	if *filterId != "" {
		records = filterRecordsById(records, *filterId)
	}

	jobs := make(chan Record, len(records))

	numWorkers := 20
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for record := range jobs {
				if err := sendRecord(record); err != nil {
					log.Printf("Error sending record %s: %v", record.ID, err)
				} else {
					fmt.Printf("Record %s sent successfully.\n", record.ID)
				}
			}
		}()
	}

	for _, record := range records {
		jobs <- record
	}
	close(jobs)

	wg.Wait()

	fmt.Printf("Processed %d records.\n", len(records))
	if len(skippedRecords) > 0 {
		fmt.Printf("Skipped %d records due to incorrect field counts:\n", len(skippedRecords))
		for _, lineNum := range skippedRecords {
			fmt.Printf(" - Line %d\n", lineNum)
		}
	}
}

func readCSV(filePath string) ([]Record, []int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1 // Allow variable field counts

	var records []Record
	var skippedRecords []int
	lineNumber := 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		lineNumber++
		if lineNumber == 1 {
			// Skip the header row
			continue
		}
		if err != nil {
			fmt.Printf("Error reading line %d: %v\n", lineNumber, err)
			skippedRecords = append(skippedRecords, lineNumber)
			continue
		}

		if len(row) != 6 {
			fmt.Printf("Skipping line %d due to incorrect number of fields (expected 6, got %d)\n", lineNumber, len(row))
			skippedRecords = append(skippedRecords, lineNumber)
			continue
		}

		category := sanitizeCategory(row[5])

		if category == "" {
			fmt.Printf("Skipping line %d due to blank category field\n", lineNumber)
			skippedRecords = append(skippedRecords, lineNumber)
			continue
		}

		records = append(records, Record{
			ID:         row[0],
			AssetName:  row[1],
			IP:         row[2],
			CreatedUTC: row[3],
			Source:     row[4],
			Category:   category,
		})
	}

	return records, skippedRecords, nil
}

func filterRecordsByCategory(records []Record, category string) []Record {
	var filtered []Record
	for _, record := range records {
		if strings.EqualFold(record.Category, category) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func filterRecordsById(records []Record, id string) []Record {
	var filtered []Record
	for _, record := range records {
		if strings.EqualFold(record.ID, id) {
			filtered = append(filtered, record)
		}
	}
	return filtered
}

func sendRecord(record Record) error {
	url := "http://localhost:8081/process"
	jsonData := fmt.Sprintf(`{"id":"%s", "asset_name":"%s", "ip":"%s", "created_utc":"%s", "source":"%s", "category":"%s"}`,
		record.ID, record.AssetName, record.IP, record.CreatedUTC, record.Source, record.Category)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(jsonData)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send record, status: %d", resp.StatusCode)
	} else {
		fmt.Printf("Record %s sent successfully.\n", record.ID)
	}
	return nil
}
