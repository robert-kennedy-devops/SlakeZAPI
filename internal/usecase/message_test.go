package usecase

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/whatsapp-saas/api/internal/domain"
)

func TestEnrichMediaRequestFillsMimeAndFilename(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("fake-image-bytes"))
	}))
	defer srv.Close()

	req, err := enrichMediaRequest(context.Background(), domain.SendMediaMessageRequest{
		Phone: "5511999999999",
		Type:  "image",
		URL:   srv.URL + "/avatar",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.MimeType != "image/png" {
		t.Fatalf("expected mime type image/png, got %q", req.MimeType)
	}
	if req.FileName != "avatar" {
		t.Fatalf("expected inferred file name avatar, got %q", req.FileName)
	}
	if len(req.Data) == 0 {
		t.Fatal("expected media bytes to be loaded")
	}
}
