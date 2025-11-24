package util

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/File-Sharing-BondBridg/File-Service/internal/services"
	clamd "github.com/dutchcoders/go-clamd"
)

func ScanFile(fileID, objectName, clamAvUrl string) {
	minioService := services.GetMinioService()
	tempPath := fmt.Sprintf("/tmp/%s", objectName)

	// Download from MinIO for scanning
	if err := minioService.DownloadFile(objectName, tempPath); err != nil {
		log.Println("Failed to download for scanning:", err)
		return
	}
	defer os.Remove(tempPath)

	// Connect to ClamAV
	c := clamd.NewClamd(clamAvUrl)
	response, err := c.ScanFile(tempPath)
	if err != nil {
		log.Println("Scan failed:", err)
		return
	}

	status := "clean"
	for res := range response {
		if res.Status == clamd.RES_FOUND {
			log.Printf("Virus detected in %s: %s", fileID, res.Description)
			status = "infected"

			// delete infected file
			err := minioService.DeleteFile(objectName)
			if err != nil {
				log.Println("Failed to delete infected file:", err)
				return
			}
		}
	}

	// Update metadata
	if err := services.UpdateFileScanStatus(fileID, status, time.Now()); err != nil {
		log.Println("Failed to update scan status:", err)
	} else {
		log.Printf("Scan finished for %s: %s", fileID, status)
	}
}
