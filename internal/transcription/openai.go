package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/whatsapp-saas/api/internal/domain"
)

const maxTranscriptionUploadSize = 25 << 20

type OpenAITranscriber struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewOpenAITranscriber(apiKey, model string) *OpenAITranscriber {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini-transcribe"
	}
	return &OpenAITranscriber{
		apiKey:     apiKey,
		model:      model,
		httpClient: &http.Client{},
	}
}

func (t *OpenAITranscriber) Transcribe(ctx context.Context, fileName, mimeType string, data []byte) (*domain.MessageTranscript, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: audio payload is empty", domain.ErrBadRequest)
	}

	uploadName, uploadMime, uploadData, err := prepareAudioForTranscription(ctx, fileName, mimeType, data)
	if err != nil {
		return nil, err
	}
	if len(uploadData) > maxTranscriptionUploadSize {
		return nil, fmt.Errorf("%w: audio file exceeds 25 MB upload limit", domain.ErrBadRequest)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("model", t.model); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	if err := writer.WriteField("response_format", "json"); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	part, err := writer.CreateFormFile("file", uploadName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	if _, err := part.Write(uploadData); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if uploadMime != "" {
		req.Header.Set("X-Upload-Content-Type", uploadMime)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%w: OpenAI returned status %d: %s", domain.ErrMessageTranscriptFailed, resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	if strings.TrimSpace(payload.Text) == "" {
		return nil, fmt.Errorf("%w: empty transcript", domain.ErrMessageTranscriptFailed)
	}

	return &domain.MessageTranscript{
		Text:     strings.TrimSpace(payload.Text),
		Provider: "openai",
		Model:    t.model,
	}, nil
}

func prepareAudioForTranscription(ctx context.Context, fileName, mimeType string, data []byte) (string, string, []byte, error) {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "audio"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		if guessed := extensionFromMimeType(mimeType); guessed != "" {
			ext = guessed
			name += guessed
		}
	}

	if isSupportedTranscriptionExtension(ext) {
		return name, normalizeAudioMimeType(mimeType, ext), data, nil
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", "", nil, fmt.Errorf("%w: ffmpeg is required to transcode %s audio", domain.ErrMessageTranscriptOff, ext)
	}

	inputFile, err := os.CreateTemp("", "slakezapi-audio-*"+ext)
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	inputPath := inputFile.Name()
	defer os.Remove(inputPath)
	if _, err := inputFile.Write(data); err != nil {
		inputFile.Close()
		return "", "", nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	if err := inputFile.Close(); err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}

	outputFile, err := os.CreateTemp("", "slakezapi-audio-*.wav")
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		"-y",
		"-i", inputPath,
		"-vn",
		"-acodec", "pcm_s16le",
		"-ar", "16000",
		"-ac", "1",
		outputPath,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", "", nil, fmt.Errorf("%w: ffmpeg conversion failed: %s", domain.ErrMessageTranscriptFailed, strings.TrimSpace(stderr.String()))
	}

	converted, err := os.ReadFile(outputPath)
	if err != nil {
		return "", "", nil, fmt.Errorf("%w: %v", domain.ErrMessageTranscriptFailed, err)
	}

	base := strings.TrimSuffix(name, filepath.Ext(name))
	if base == "" {
		base = "audio"
	}
	return base + ".wav", "audio/wav", converted, nil
}

func isSupportedTranscriptionExtension(ext string) bool {
	switch ext {
	case ".mp3", ".mp4", ".mpeg", ".mpga", ".m4a", ".wav", ".webm":
		return true
	default:
		return false
	}
}

func extensionFromMimeType(mimeType string) string {
	base, _, err := mime.ParseMediaType(mimeType)
	if err != nil {
		base = mimeType
	}
	switch strings.ToLower(strings.TrimSpace(base)) {
	case "audio/mpeg":
		return ".mp3"
	case "audio/mp4":
		return ".mp4"
	case "audio/x-m4a", "audio/m4a":
		return ".m4a"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	case "audio/webm":
		return ".webm"
	case "video/mp4":
		return ".mp4"
	case "audio/ogg", "application/ogg":
		return ".ogg"
	default:
		return ""
	}
}

func normalizeAudioMimeType(mimeType, ext string) string {
	base, _, err := mime.ParseMediaType(mimeType)
	if err == nil && strings.TrimSpace(base) != "" {
		return base
	}
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "audio/mp4"
	case ".m4a":
		return "audio/m4a"
	case ".wav":
		return "audio/wav"
	case ".webm":
		return "audio/webm"
	default:
		return "application/octet-stream"
	}
}
