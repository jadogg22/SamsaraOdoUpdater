package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type Asset struct {
	ID       string
	TS       string
	Lat      string
	Lon      string
	DriverID string
	Odometer float64
}

// ****** This is the handler funtion that manages a single asset *********
// ****** by getting the
func makeMeterChange(asset MyVehicleData, accessToken string) error {

	//fmt.Println("Changing Asset: " + asset.Name)

	// Get MeterAssociationID from dossier
	meterAssociationID, err := getMeterAssociationID(asset, accessToken)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Use MeterAssociationID to change the meter reading to the new values
	err = changeMeter(asset, meterAssociationID, accessToken)
	if err != nil {
		return err
	}

	return nil
}

func getMeterAssociationID(asset MyVehicleData, accessToken string) (string, error) {
	// Make a request to retrieve meter reading information

	response, err := makeMeterReadingRequest(accessToken, asset.Name)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Check if the request was successful
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get meter reading: %s", response.Status)
	}

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var meterData []map[string]interface{}
	json.Unmarshal(body, &meterData)

	assetID := asset.Name

	// Iterate over meterData to find matching MeterAssociationID
	var matchingMeterAssociationID int
	for _, data := range meterData {
		if physicalMeter, ok := data["physicalMeter"].(map[string]interface{}); ok {
			if meterAssociation, ok := physicalMeter["meterAssociation"].(map[string]interface{}); ok {
				if asset, ok := meterAssociation["asset"].(map[string]interface{}); ok {
					if primaryAssetIdentifier, ok := asset["primaryAssetIdentifier"].(string); ok && primaryAssetIdentifier == assetID {
						if meterAssociationID, ok := meterAssociation["meterAssociationId"].(float64); ok {
							matchingMeterAssociationID = int(meterAssociationID)
							break
						}
					}
				}
			}
		}
	}

	return fmt.Sprint(matchingMeterAssociationID), nil
}

func changeMeter(asset MyVehicleData, meterAssociationID string, access_token string) error {
	// Create operation payload
	params, err := createOperation(meterAssociationID, asset.Odometer, asset.Timestamp, asset.Latitude, asset.Longitude)
	if err != nil {
		fmt.Println("Error creating operation payload:", err)
		return err
	}

	// Make meter change request
	resp, err := makeMeterChangeRequest(access_token, params)
	if err != nil {
		fmt.Println("Error making meter change request:", err)
		return err
	}

	// Check response status code
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	return nil
}

func createParams(omniAssetID string) string {
	decoded := `{"page":1,"amount":10,"orderBy":[{"field":"physicalMeter.meterAssociation.asset.primaryAssetIdentifier","dir":"asc"},{"field":"readingTime","dir":"desc"},{"field":"reading","dir":"desc"}],"filter":{"logic":"and","filters":[{"field":"physicalMeter.meterAssociation.asset.disposition.status.name","operator":"eq","value":"Active"},{"field":"physicalMeter.meterAssociation.asset.primaryAssetIdentifier","operator":"eq","value":"` + omniAssetID + `","alternateValue":null}]},"expands":[{"name":"Person"},{"name":"PhysicalMeter","expands":[{"name":"MeterAssociation","expands":[{"name":"Asset","expands":[{"name":"AssetType"}]},{"name":"Meter","expands":[{"name":"MeterTypeMeasure","expands":[{"name":"Measure"}]}]}]}]}],"groupBy":[],"aggregates":[],"globalAggregates":[]}`
	decodedBytes := []byte(decoded)
	encoded := base64.StdEncoding.EncodeToString(decodedBytes)
	return encoded
}

func makeMeterReadingRequest(accessToken, omniAssetID string) (*http.Response, error) {
	url := "https://d7.dossierondemand.com/api/assets/meterreadings"
	params := createParams(omniAssetID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.URL.RawQuery = "operation=" + params

	// Create a custom transport that skips TLS certificate verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Create an http.Client with the custom transport
	client := &http.Client{Transport: tr}

	// Perform the HTTP request
	return client.Do(req)
}

// Define the structure for the operation payload
type OperationPayload struct {
	MeterReadingID     int     `json:"meterReadingId"`
	ReadingTime        string  `json:"readingTime"`
	LifeTotal          int     `json:"lifeTotal"`
	PersonID           int     `json:"personId"`
	Description        string  `json:"description"`
	Latitude           string  `json:"latitude"`
	Longitude          string  `json:"longitude"`
	Suspect            bool    `json:"suspect"`
	SuspectApproved    *bool   `json:"suspectApproved"`
	WorkOrderID        *int    `json:"workOrderId"`
	InspectionID       *int    `json:"inspectionId"`
	FluidUsageID       *int    `json:"fluidUsageId"`
	MeterAssociationID string  `json:"meterAssociationId"`
	MeterMethodID      int     `json:"meterMethodId"`
	Reading            float64 `json:"reading"`
}

// Define the function to create the operation payload
func createOperation(meterAssociationID string, odometer float64, date string, lat, lon float64) ([]byte, error) {
	// convert lat and long into a strings
	lata := strconv.FormatFloat(lat, 'f', -1, 64)
	long := strconv.FormatFloat(lon, 'f', -1, 64)
	payload := OperationPayload{
		MeterReadingID:     0,
		ReadingTime:        date,
		LifeTotal:          0,
		PersonID:           6438,
		Description:        "created from API 3.0",
		Latitude:           lata,
		Longitude:          long,
		Suspect:            false,
		SuspectApproved:    nil,
		WorkOrderID:        nil,
		InspectionID:       nil,
		FluidUsageID:       nil,
		MeterAssociationID: meterAssociationID,
		MeterMethodID:      1,
		Reading:            odometer,
	}

	jsonParams, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return jsonParams, nil
}

// Define the function to make the meter change request without verifying SSL certificates
func makeMeterChangeRequest(accessToken string, params []byte) (*http.Response, error) {
	url := "https://d7.dossierondemand.com/api/assets/meterreadings/CreateMeterReading"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(params))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Skip SSL certificate verification
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return resp, nil
}
