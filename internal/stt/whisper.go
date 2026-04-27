package stt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

const defaultBaseURL = "https://api.openai.com/v1"

type WhisperProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

func New(apiKey string) (*WhisperProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("stt: api key is required")
	}
	return &WhisperProvider{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		client:  &http.Client{},
	}, nil
}

// SetBaseURL overrides the API base URL. Intended for testing only.
func (w *WhisperProvider) SetBaseURL(u string) { w.baseURL = u }

func (w *WhisperProvider) Transcribe(ctx context.Context, audio []byte) (string, error) {
	body, contentType, err := buildMultipart(audio)
	if err != nil {
		return "", fmt.Errorf("stt transcribe: build request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.baseURL+"/audio/transcriptions", body)
	if err != nil {
		return "", fmt.Errorf("stt transcribe: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("stt transcribe: http: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("stt transcribe: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stt transcribe: api status %d: %s", resp.StatusCode, raw)
	}

	return strings.TrimSpace(string(raw)), nil
}

func buildMultipart(audio []byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("file", "audio.wav")
	if err != nil {
		return nil, "", err
	}
	if _, err = fw.Write(audio); err != nil {
		return nil, "", err
	}
	if err = mw.WriteField("model", "whisper-1"); err != nil {
		return nil, "", err
	}
	if err = mw.WriteField("response_format", "text"); err != nil {
		return nil, "", err
	}
	if err = mw.Close(); err != nil {
		return nil, "", err
	}
	return &buf, mw.FormDataContentType(), nil
}
