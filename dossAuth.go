package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// GetAccessTokenWithCred retrieves an access token using provided credentials.
func GetAccessTokenWithCred(username string, password string, secret string) (string, error) {

	APIURL := "https://authentication.d7.dossierondemand.com/connect/token"

	// Create an HTTP client with TLS certificate verification disabled
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{Transport: transport}

	response, err := client.PostForm(APIURL, url.Values{
		"grant_type":    {"password"},
		"username":      {username},
		"password":      {password},
		"scope":         {"DossierApi"},
		"client_id":     {"sharpTransportationClient"},
		"client_secret": {secret}})

	//okay, moving on...
	if err != nil {
		//handle postform error
		fmt.Println("error buddy")
		return "", err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)

	if err != nil {
		//handle read response error
		fmt.Println("there rwas an err")
	}

	var accessTokenResponse AccessTokenResponse
	if err := json.Unmarshal([]byte(body), &accessTokenResponse); err != nil {
		fmt.Println("Error:", err)
		return "", nil
	}

	return accessTokenResponse.AccessToken, nil
}

// GetAccessToken retrieves an access token using environment variables.
func GetAccessToken() (string, error) {
	username := os.Getenv("dossUsername")
	password := os.Getenv("dossPassword")
	secret := os.Getenv("client_secret")

	if username == "" || password == "" || secret == "" {
		return "", fmt.Errorf("missing environment variables")
	}

	return GetAccessTokenWithCred(username, password, secret)
}
