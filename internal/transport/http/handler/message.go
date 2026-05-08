package handler

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/whatsapp-saas/api/internal/domain"
	"github.com/whatsapp-saas/api/internal/middleware"
	"github.com/whatsapp-saas/api/internal/usecase"
	"github.com/whatsapp-saas/api/pkg/httputil"
	"github.com/whatsapp-saas/api/pkg/logger"
)

// MessageHandler exposes messaging endpoints.
type MessageHandler struct {
	msgUC *usecase.MessageUsecase
	log   *logger.Logger
}

func NewMessageHandler(msgUC *usecase.MessageUsecase, log *logger.Logger) *MessageHandler {
	return &MessageHandler{msgUC: msgUC, log: log}
}

// Send godoc
// POST /messages/send
// Header: Authorization: Bearer <api_key>
// Body: {"phone": "5511999999999", "message": "Hello!"}
func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.SendMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send", map[string]interface{}{
		"message_id": resp.MessageID,
		"phone":      req.Phone,
		"type":       "text",
	})

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendMedia(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.SendMediaMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendMediaMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send_media", map[string]interface{}{
		"message_id": resp.MessageID,
		"phone":      req.Phone,
		"type":       req.Type,
		"mime_type":  req.MimeType,
	})

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendInteractive(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.InteractiveMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendInteractiveMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendGroup(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.GroupMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendGroupMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) StatusPost(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.StatusMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.PostStatus(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	resp, err := h.msgUC.ListGroups(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()))
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	resp, err := h.msgUC.ListContacts(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()))
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) ResolveContacts(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.ResolveContactsRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.ResolveContacts(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("contacts.resolve", map[string]interface{}{
		"count": len(resp),
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendBulk(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	var req domain.BulkSendMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendBulkMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send_bulk", map[string]interface{}{
		"total":    resp.Total,
		"accepted": resp.Accepted,
		"sent":     resp.Sent,
		"failed":   resp.Failed,
	})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.msgUC.ListMessages(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()), r.URL.Query())
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	resp, err := h.msgUC.ListConversations(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()), r.URL.Query())
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) UpdateConversation(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.UpdateConversationRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := h.msgUC.UpdateConversation(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()), r.PathValue("phone"), req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendLocation(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.LocationMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendLocationMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send_location", map[string]interface{}{"message_id": resp.MessageID, "phone": req.Phone})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendContactCard(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.ContactCardRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendContactCard(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send_contact", map[string]interface{}{"message_id": resp.MessageID, "phone": req.Phone})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) SendSticker(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.SendMediaMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendSticker(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send_sticker", map[string]interface{}{"message_id": resp.MessageID, "phone": req.Phone})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) React(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.ReactMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	if err := h.msgUC.ReactToMessage(r.Context(), tenantID, req); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessageHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.DeleteMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	if err := h.msgUC.DeleteMessage(r.Context(), tenantID, req); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *MessageHandler) SendQuoted(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.QuotedSendRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.SendQuotedMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.send_quoted", map[string]interface{}{"message_id": resp.MessageID, "phone": req.Phone})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) Edit(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.EditMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	if err := h.msgUC.EditMessage(r.Context(), tenantID, req); err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.edit", map[string]interface{}{"message_id": req.MessageID})
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessageHandler) Forward(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.ForwardMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	resp, err := h.msgUC.ForwardMessage(r.Context(), tenantID, req)
	if err != nil {
		httputil.DomainError(w, err)
		return
	}
	h.log.WithContext(r.Context()).Audit("message.forward", map[string]interface{}{"message_id": resp.MessageID, "phone": req.Phone})
	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) Star(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}
	var req domain.StarMessageRequest
	if err := httputil.Decode(r, &req); err != nil {
		httputil.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.InstanceID = middleware.InstanceFromCtx(r.Context())
	if err := h.msgUC.StarMessage(r.Context(), tenantID, req); err != nil {
		httputil.DomainError(w, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessageHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.msgUC.GetMessage(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()), r.PathValue("id"))
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	httputil.JSON(w, http.StatusOK, resp)
}

func (h *MessageHandler) GetMedia(w http.ResponseWriter, r *http.Request) {
	tenantID := middleware.TenantFromCtx(r.Context())
	if tenantID == "" {
		httputil.Error(w, http.StatusUnauthorized, "missing tenant context")
		return
	}

	resp, err := h.msgUC.GetMessageMedia(r.Context(), tenantID, middleware.InstanceFromCtx(r.Context()), r.PathValue("id"))
	if err != nil {
		httputil.DomainError(w, err)
		return
	}

	if resp.MimeType != "" {
		w.Header().Set("Content-Type", resp.MimeType)
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	if resp.FileName != "" {
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": resp.FileName}))
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(resp.Data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.Data)
}
