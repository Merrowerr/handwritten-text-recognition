package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type TokenRequest struct {
	YandexPassportOauthToken string `json:"yandexPassportOauthToken"`
}

type TokenResponse struct {
	IamToken  string `json:"iamToken"`
	ExpiresAt string `json:"expiresAt"`
}

func getIAMToken(oauthToken string) (string, error) {
	url := "https://iam.api.cloud.yandex.net/iam/v1/tokens"

	// Create request body
	requestBody, err := json.Marshal(TokenRequest{
		YandexPassportOauthToken: oauthToken,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %v", err)
	}

	return tokenResp.IamToken, nil
}

func UpdateIAM() {
	for {
		iam, err := getIAMToken(os.Getenv("YANDEX_OAUTH"))
		if err != nil {
			UpdateIAM()
		}
		os.Setenv("IAM_TOKEN", iam)
		time.Sleep(12 * time.Hour)
	}
}
