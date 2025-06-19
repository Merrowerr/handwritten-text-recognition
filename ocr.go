package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// OCRResponse defines the structure for Yandex OCR API responses.
type OCRResponse struct {
	Result struct {
		TextAnnotation struct {
			Width  string `json:"width"`
			Height string `json:"height"`
			Blocks []struct {
				BoundingBox struct {
					Vertices []struct {
						X string `json:"x"`
						Y string `json:"y"`
					} `json:"vertices"`
				} `json:"bounding_box"`
				Lines []struct {
					Words []struct {
						Text string `json:"text"`
					} `json:"words"`
				} `json:"lines"`
			} `json:"blocks"`
			FullText string `json:"fullText"`
		} `json:"textAnnotation"`
	} `json:"result"`
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// MistralResponse defines the structure for Mistral Chat API responses.
type MistralResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
			Role    string `json:"role"`
		} `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Timing tracks the duration of OCR, Mistral API, and total processing.
type Timing struct {
	OCRTime   float64
	GPTTime   float64
	TotalTime float64
}

// logTiming writes timing metrics to a log file.
func logTiming(timing Timing) {
	logEntry := fmt.Sprintf(
		"[%s] OCR: %.2f sec, Mistral: %.2f sec, Total: %.2f sec\n",
		time.Now().Format(time.RFC3339),
		timing.OCRTime,
		timing.GPTTime,
		timing.TotalTime,
	)
	f, err := os.OpenFile("timing.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening timing.log: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(logEntry); err != nil {
		fmt.Printf("Error writing to timing.log: %v\n", err)
	}
}

// YandexOCR performs OCR on an image using the Yandex OCR API.
func YandexOCR(imagePath, folderID, iamToken string) (string, float64, error) {
	start := time.Now()
	url := "https://ocr.api.cloud.yandex.net/ocr/v1/recognizeText"

	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		return "", 0, fmt.Errorf("check file: %v", err)
	}
	if fileInfo.Size() == 0 {
		return "", 0, fmt.Errorf("image file is empty")
	}

	file, err := os.Open(imagePath)
	if err != nil {
		return "", 0, fmt.Errorf("open image: %v", err)
	}
	defer file.Close()

	imgBytes, err := io.ReadAll(file)
	if err != nil {
		return "", 0, fmt.Errorf("read image: %v", err)
	}
	if len(imgBytes) == 0 {
		return "", 0, fmt.Errorf("image data is empty")
	}
	imgBase64 := base64.StdEncoding.EncodeToString(imgBytes)
	if imgBase64 == "" {
		return "", 0, fmt.Errorf("base64 encoding failed")
	}

	payload := map[string]interface{}{
		"mimeType":      "image/jpeg",
		"languageCodes": []string{"ru"},
		"model":         "handwritten",
		"content":       imgBase64,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal payload: %v", err)
	}

	fmt.Printf("OCR Request Body: %s\n", string(body))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", 0, fmt.Errorf("create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+iamToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-folder-id", folderID)
	req.Header.Set("x-data-logging-enabled", "true")

	client := &http.Client{
		Timeout: 30 * time.Second, // Increased timeout for reliability
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("OCR failed: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("read response: %v", err)
	}

	if err := os.WriteFile("api_response.json", respBody, 0644); err != nil {
		return "", 0, fmt.Errorf("write api_response.json: %v", err)
	}

	var ocrResp OCRResponse
	if err := json.Unmarshal(respBody, &ocrResp); err != nil {
		return "", 0, fmt.Errorf("unmarshal response: %v", err)
	}

	if ocrResp.Error.Message != "" {
		return "", time.Since(start).Seconds(), fmt.Errorf("OCR error: %s", ocrResp.Error.Message)
	}

	text := ocrResp.Result.TextAnnotation.FullText
	if text == "" {
		for _, block := range ocrResp.Result.TextAnnotation.Blocks {
			for _, line := range block.Lines {
				for _, word := range line.Words {
					text += word.Text + " "
				}
				text += "\n"
			}
		}
	}

	if text == "" {
		return "", time.Since(start).Seconds(), fmt.Errorf("empty text detected")
	}

	return strings.TrimSpace(text), time.Since(start).Seconds(), nil
}

// checkIP verifies the public IP address, with or without a proxy.
func checkIP(useProxy bool, proxyAddr string) (string, error) {
	url := "https://api.ipify.org?format=text"

	var client *http.Client
	if useProxy {
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return "", fmt.Errorf("failed to setup SOCKS5 proxy: %v", err)
		}
		transport := &http.Transport{
			Dial: dialer.Dial,
		}
		client = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	return string(body), nil
}

// logIP logs the IP check results.
func logIP(ip string, useProxy bool, proxyAddr string, err error) {
	logEntry := fmt.Sprintf(
		"[%s] Proxy: %v, ProxyAddr: %s, IP: %s, Error: %v\n",
		time.Now().Format(time.RFC3339),
		useProxy,
		proxyAddr,
		ip,
		err,
	)
	fmt.Print(logEntry)
	f, err := os.OpenFile("proxy_check.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening proxy_check.log: %v\n", err)
		return
	}
	defer f.Close()
	if _, err := f.WriteString(logEntry); err != nil {
		fmt.Printf("Error writing to proxy_check.log: %v\n", err)
	}
}

// MistralAPI interacts with the Mistral Chat API to correct OCR text.
// Requires MISTRAL_API_KEY environment variable.
// On Windows, set DNS to 8.8.8.8 or 1.1.1.1 if DNS resolution fails (Control Panel > Network > Adapter > IPv4 > DNS).
func MistralAPI(text, apiKey string) (string, float64, error) {
	start := time.Now()
	url := "https://api.mistral.ai/v1/chat/completions"

	// Model and proxy settings from environment variables
	model := os.Getenv("MISTRAL_MODEL")
	if model == "" {
		model = "mistral-large-latest" // Default per Mistral API docs
	}

	useProxy := os.Getenv("USE_PROXY") == "true" // Default to false unless explicitly true
	proxyAddr := os.Getenv("PROXY_ADDR")
	if proxyAddr == "" {
		proxyAddr = "127.0.0.1:10808"
	}

	// Check IP for debugging network issues
	fmt.Printf("Checking IP (Proxy: %v, Addr: %s)...\n", useProxy, proxyAddr)
	ip, err := checkIP(useProxy, proxyAddr)
	logIP(ip, useProxy, proxyAddr, err)
	if err != nil {
		fmt.Printf("IP check failed: %v\n", err)
	}

	// Configure HTTP client
	var client *http.Client
	if useProxy {
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return "", 0, fmt.Errorf("setup SOCKS5 proxy: %v", err)
		}
		transport := &http.Transport{
			Dial: dialer.Dial,
		}
		client = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second, // Increased for reliability
		}
	} else {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	// Construct payload per Mistral API specs
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": fmt.Sprintf(
					"Исправьте ошибки OCR в тексте, сохраняя оригинальный язык и переносы строк. Исправляйте ТОЛЬКО явные орфографические ошибки или неполные слова на основе написания и контекста. Не добавляйте и не удаляйте слова, не изменяйте структуру, порядок слов, пунктуацию, смысл и самое главное - переносы строк, даже если текст нелогичен. Сводите исправления к минимуму. Возвращайте только исправленный текст без дополнительных комментариев. если текст довольно неразборчивый, в самом конце добавляй текст \"слишком неразборчиво 9905148\".\n\n%s",
					text,
				),
			},
		},
		"temperature": 0.3,
		"max_tokens":  2000,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", 0, fmt.Errorf("marshal payload: %v", err)
	}
	fmt.Printf("Mistral Request Body: %s\n", string(body))

	// Retry logic for transient network issues
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		fmt.Printf("Mistral API attempt %d/%d at %s\n", attempt, maxRetries, time.Now().Format(time.RFC3339))
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
		if err != nil {
			return "", 0, fmt.Errorf("create request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Attempt %d failed: %v\n", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
				continue
			}
			return "", 0, fmt.Errorf("send request after %d attempts: %v", maxRetries, err)
		}
		defer resp.Body.Close()

		fmt.Printf("Mistral Response Status: %d\n", resp.StatusCode)
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Attempt %d failed to read response: %v\n", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return "", 0, fmt.Errorf("read response after %d attempts: %v", maxRetries, err)
		}

		// Save response for debugging
		responseFile := filepath.Join("mistral_response.json")
		if err := os.WriteFile(responseFile, respBody, 0644); err != nil {
			fmt.Printf("Error writing %s: %v\n", responseFile, err)
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Attempt %d failed with status: %d, body: %s\n", attempt, resp.StatusCode, string(respBody))
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return "", 0, fmt.Errorf("Mistral failed: status %d, body: %s", resp.StatusCode, string(respBody))
		}

		var mistralResp MistralResponse
		if err := json.Unmarshal(respBody, &mistralResp); err != nil {
			fmt.Printf("Attempt %d failed to unmarshal: %v\n", attempt, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return "", 0, fmt.Errorf("unmarshal response after %d attempts: %v", maxRetries, err)
		}

		if mistralResp.Error.Message != "" {
			return "", time.Since(start).Seconds(), fmt.Errorf("Mistral error: %s (type: %s)", mistralResp.Error.Message, mistralResp.Error.Type)
		}

		if len(mistralResp.Choices) == 0 || mistralResp.Choices[0].Message.Content == "" {
			fmt.Printf("Attempt %d: no valid response, body: %s\n", attempt, string(respBody))
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return "", time.Since(start).Seconds(), fmt.Errorf("no Mistral response, body: %s", string(respBody))
		}

		return strings.TrimSpace(mistralResp.Choices[0].Message.Content), time.Since(start).Seconds(), nil
	}

	return "", 0, fmt.Errorf("Mistral request failed after %d attempts", maxRetries)
}

// ProcessImage orchestrates OCR and Mistral API processing.
func ProcessImage(imagePath, folderID, iamToken, mistralAPIKey string) (string, string, Timing, error) {
	startTotal := time.Now()
	ocrText, ocrTime, err := YandexOCR(imagePath, folderID, iamToken)
	if err != nil {
		return "", "", Timing{OCRTime: ocrTime}, fmt.Errorf("OCR: %v", err)
	}
	gptText, gptTime, err := MistralAPI(ocrText, mistralAPIKey)
	if err != nil {
		return ocrText, "", Timing{OCRTime: ocrTime, GPTTime: gptTime}, fmt.Errorf("Mistral: %v", err)
	}
	totalTime := time.Since(startTotal).Seconds()
	timing := Timing{OCRTime: ocrTime, GPTTime: gptTime, TotalTime: totalTime}
	logTiming(timing)
	return ocrText, gptText, timing, nil
}
