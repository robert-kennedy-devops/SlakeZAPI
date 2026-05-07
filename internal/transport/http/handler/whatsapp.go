package handler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/skip2/go-qrcode"
	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// WhatsAppHandler exposes WhatsApp session endpoints.
type WhatsAppHandler struct {
	waUC *usecase.WhatsAppUsecase
	log  *logger.Logger
}

func NewWhatsAppHandler(waUC *usecase.WhatsAppUsecase, log *logger.Logger) *WhatsAppHandler {
	return &WhatsAppHandler{waUC: waUC, log: log}
}

// Connect godoc
// POST /whatsapp/connect
// Header: Authorization: Bearer <api_key>
// Response: {"qr_code": "...", "status": "connecting"}
func (h *WhatsAppHandler) Connect(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.waUC.Connect(r.Context(), tenantID, instanceID)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("whatsapp.connect", map[string]interface{}{
		"status": resp.Status,
		"phone":  resp.Phone,
	})
	attachQRLinks(r, resp)

	httputil.JSON(w, http.StatusOK, resp)
}

// Status godoc
// GET /whatsapp/status
func (h *WhatsAppHandler) Status(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	session, err := h.waUC.GetSession(r.Context(), tenantID, instanceID)
	if err == nil {
		if session.QRCode != "" {
			resp := &struct {
				*domain.Session
				QRPNGURL  string `json:"qr_png_url,omitempty"`
				QRPageURL string `json:"qr_page_url,omitempty"`
			}{
				Session:   session,
				QRPNGURL:  absoluteURL(r, whatsappPath(r, "/qr.png")),
				QRPageURL: absoluteURL(r, whatsappPath(r, "/qr")),
			}
			httputil.JSON(w, http.StatusOK, resp)
			return
		}
		httputil.JSON(w, http.StatusOK, session)
		return
	}

	status := h.waUC.GetStatus(r.Context(), tenantID, instanceID)
	httputil.JSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

func (h *WhatsAppHandler) QRPage(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	session, err := h.waUC.GetSession(r.Context(), tenantID, instanceID)
	if err != nil || session.QRCode == "" {
		httputil.Error(w, http.StatusNotFound, "qr code not available")
		return
	}

	pngURL := absoluteURL(r, whatsappPath(r, "/qr.png"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<!doctype html><html><head><meta charset=\"utf-8\"><title>WhatsApp QR</title><style>body{font-family:sans-serif;padding:24px;max-width:720px;margin:auto}img{width:320px;height:320px;border:1px solid #ddd;padding:12px;background:#fff}code,pre{white-space:pre-wrap;word-break:break-all}</style></head><body><h1>WhatsApp QR</h1><p>Status: <strong>%s</strong></p><p>Escaneie este QR no WhatsApp em Dispositivos conectados.</p><p><img src=\"%s\" alt=\"WhatsApp QR\"></p><p>Se falhar, consulte <code>/whatsapp/status</code> para ver <code>last_error</code> e gere um novo QR com <code>POST /whatsapp/connect</code>.</p><h2>QR bruto</h2><pre>%s</pre></body></html>", session.Status, pngURL, session.QRCode)
}

func (h *WhatsAppHandler) QRPNG(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	session, err := h.waUC.GetSession(r.Context(), tenantID, instanceID)
	if err != nil || session.QRCode == "" {
		httputil.Error(w, http.StatusNotFound, "qr code not available")
		return
	}

	png, err := qrcode.Encode(session.QRCode, qrcode.Medium, 320)
	if err != nil {
		h.log.WithContext(r.Context()).Error("failed to encode whatsapp qr", map[string]interface{}{"err": err.Error()})
		httputil.Error(w, http.StatusInternalServerError, "failed to encode qr code")
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(png)
}

func (h *WhatsAppHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	if err := h.waUC.Disconnect(r.Context(), tenantID, instanceID); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("whatsapp.disconnect")

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "disconnected"})
}

func (h *WhatsAppHandler) Logout(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	instanceID := middleware.InstanceFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	if err := h.waUC.Logout(r.Context(), tenantID, instanceID); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("whatsapp.logout")

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
func attachQRLinks(r *http.Request, resp *domain.ConnectResponse) {
	if resp == nil || resp.QRCode == "" {
		return
	}
	resp.QRPNGURL = absoluteURL(r, whatsappPath(r, "/qr.png"))
	resp.QRPageURL = absoluteURL(r, whatsappPath(r, "/qr"))
}

func absoluteURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	u := url.URL{
		Scheme: scheme,
		Host:   r.Host,
		Path:   path,
	}
	return u.String()
}

func whatsappPath(r *http.Request, suffix string) string {
	base := "/whatsapp"
	if strings.HasPrefix(r.URL.Path, "/app/whatsapp") {
		base = "/app/whatsapp"
	}
	return base + suffix
}
