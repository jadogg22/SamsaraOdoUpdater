package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// VehicleData represents the structure of the data in the JSON response
type VehicleData struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ExternalIDs struct {
		Serial string `json:"samsara.serial"`
		VIN    string `json:"samsara.vin"`
	} `json:"externalIds"`
	ObdOdometerMeters struct {
		Time  string  `json:"time"`
		Value float64 `json:"value"`
	} `json:"obdOdometerMeters"`
	GPS struct {
		Time              string  `json:"time"`
		Latitude          float64 `json:"latitude"`
		Longitude         float64 `json:"longitude"`
		HeadingDegrees    float64 `json:"headingDegrees"`
		SpeedMilesPerHour float64 `json:"speedMilesPerHour"`
		ReverseGeo        struct {
			FormattedLocation string `json:"formattedLocation"`
		} `json:"reverseGeo"`
	} `json:"gps"`
}

type MyVehicleData struct {
	Name      string
	Timestamp string
	Odometer  float64
	Place     string
	Latitude  float64
	Longitude float64
}

func (v MyVehicleData) Print() {
	fmt.Printf("Name: %s\n", v.Name)
	fmt.Printf("Timestamp %s\n", v.Timestamp)
	fmt.Printf("Odometer: %.2f\n", v.Odometer)
	fmt.Printf("%s ", v.Place)
	fmt.Printf("Lat: %.6f, ", v.Latitude)
	fmt.Printf("Long: %.6f\n", v.Longitude)
	fmt.Println("--------------------------")
	fmt.Println()
}

func SamsaraRequest(token string) ([]byte, error) {
	// What kind of request are we looking to create?
	baseURL, err := url.Parse("https://api.samsara.com/fleet/vehicles/stats")
	if err != nil {
		return nil, err
	}

	query := baseURL.Query()
	query.Set("types", "obdOdometerMeters,gps")
	baseURL.RawQuery = query.Encode()

	// Create the HTTP request
	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		return nil, err
	}

	// Set the Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil

}

func ParseSamsaraResponse(responseJSON []byte) ([]MyVehicleData, error) {
	var data struct {
		Data []VehicleData `json:"data"`
	}

	if err := json.Unmarshal(responseJSON, &data); err != nil {
		return nil, err
	}

	result := make([]MyVehicleData, len(data.Data))
	for i, vehicle := range data.Data {
		result[i] = MyVehicleData{
			Name:      vehicle.Name,
			Timestamp: vehicle.ObdOdometerMeters.Time,
			Odometer:  (vehicle.ObdOdometerMeters.Value / 1609.344), // change from meters to miles
			Place:     vehicle.GPS.ReverseGeo.FormattedLocation,
			Latitude:  vehicle.GPS.Latitude,
			Longitude: vehicle.GPS.Longitude,
		}
	}

	return result, nil
}
