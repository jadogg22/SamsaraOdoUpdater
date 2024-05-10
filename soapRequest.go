package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func generateRandomChar(length int) string {
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomString := make([]byte, length)
	rand.Read(randomString)
	for i := range randomString {
		randomString[i] = chars[int(randomString[i])%len(chars)]
	}
	return string(randomString)
}

func generateUsernameToken(username, password string) string {
	id := generateRandomChar(30)
	nonce := generateRandomChar(16)
	created := time.Now().UTC().Format("2006-01-02T15:04:05Z")

	nonceBase64 := base64.StdEncoding.EncodeToString([]byte(nonce))

	return fmt.Sprintf(`
        <wsse:UsernameToken wsu:Id="UsernameToken-%s" xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
            <wsse:Username>%s</wsse:Username>
            <wsse:Password Type="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-username-token-profile-1.0#PasswordText">%s</wsse:Password>
            <wsse:Nonce EncodingType="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-soap-message-security-1.0#Base64Binary">%s</wsse:Nonce>
            <wsu:Created>%s</wsu:Created>
        </wsse:UsernameToken>
    `, id, username, password, nonceBase64, created)
}

func createRequest(username, password, lastTransaction string) (string, string, error) {
	subscriberID := "7"
	transactionID := lastTransaction
	endpoint := "https://services.omnitracs.com:443/otsWebWS/services/OTSWebSvcs"

	usernameToken := generateUsernameToken(username, password)

	headers := map[string]string{
		"Content-Type":    "text/xml;charset=UTF-8",
		"SOAPAction":      "",
		"Accept-Encoding": "gzip,deflate",
		"Connection":      "Keep-Alive",
	}

	soapBody := fmt.Sprintf(`
	<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:web="http://websvcs.otswebws">
            <soapenv:Header>
                <wsse:Security soapenv:mustUnderstand="1" 
					xmlns:wsse="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd" 
					xmlns:wsu="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd">
                    %s
                </wsse:Security>
            </soapenv:Header>
            <soapenv:Body>
                <web:dequeue2>
                    <subscriberId>%s</subscriberId>
                    <transactionIdIn>%s</transactionIdIn>
                </web:dequeue2>
            </soapenv:Body>
        </soapenv:Envelope>
    `, usernameToken, subscriberID, transactionID)

	request, err := http.NewRequest("POST", endpoint, strings.NewReader(soapBody))
	if err != nil {
		return "", "", err
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", "", err
	}

	data, err1 := responseToString(response)
	if err1 != nil {
		return "", "", err1
	}

	if hasTransactions(data) {
		return extractResponseData(data)
	} else {
		return "", "", nil
	}
}

func responseToString(response *http.Response) (string, error) {
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type Envelope struct {
	Body Body `xml:"Body"`
}

type Body struct {
	Return Return `xml:"dequeue2Return"`
}

type Return struct {
	Count            string `xml:"count"`
	Transactions     string `xml:"transactions"`
	TransactionIDOut string `xml:"transactionIdOut"`
}

func extractResponseData(responseBody string) (string, string, error) {
	// Find the start and end index of the transactions part

	// Check for count
	countStart := strings.Index(string(responseBody), "<count>")
	countEnd := strings.Index(string(responseBody), "</count>")
	if countStart != -1 && countEnd != -1 {
		countStr := responseBody[countStart+len("<count>") : countEnd]
		count, err := strconv.Atoi(countStr)
		if err != nil {
			fmt.Println("Error converting count:", err)
			return "", "", err
		}
		if count == 0 {
			return "", "", err // Exit the loop if count is zero
		}
	}

	start := strings.Index(responseBody, "<transactions>")
	end := strings.Index(responseBody, "</transactions>")
	if start == -1 || end == -1 {
		return "", "", fmt.Errorf("transactions part not found in response body")
	}

	// Extract the encoded transactions data
	encodedTransactions := responseBody[start+len("<transactions>") : end]

	// Decode the encoded transactions
	decodedTransactions, err := base64.StdEncoding.DecodeString(encodedTransactions)
	if err != nil {
		return "", "", fmt.Errorf("error decoding transactions data: %v", err)
	}

	// Extract the transactionIdOut directly from the XML without decoding
	transactionIDStart := strings.Index(responseBody, "<transactionIdOut>")
	transactionIDEnd := strings.Index(responseBody, "</transactionIdOut>")
	if transactionIDStart == -1 || transactionIDEnd == -1 {
		return "", "", fmt.Errorf("transactionIdOut not found in response body")
	}
	transactionIDOut := responseBody[transactionIDStart+len("<transactionIdOut>") : transactionIDEnd]

	// Return the extracted data
	return string(decodedTransactions), transactionIDOut, nil
}

func hasTransactions(responseBody string) bool {
	// Find the start and end index of the count element
	countStart := strings.Index(responseBody, "<count>")
	countEnd := strings.Index(responseBody, "</count>")
	if countStart == -1 || countEnd == -1 {
		return false // count element not found, assume no transactions
	}

	// Extract the count value
	countStr := responseBody[countStart+len("<count>") : countEnd]
	count, err := strconv.Atoi(countStr)
	if err != nil {
		fmt.Println("error converting count")
		return false // Error converting count, assume no transactions
	}

	// Return true if count is greater than zero, false otherwise
	return count > 0
}
