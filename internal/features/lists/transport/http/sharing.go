package lists_transport_http

import (
	"net/http"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

// CreateInvite 	godoc
// @Summary 		Ссылка-приглашение в список
// @Description 	Выдаёт новый код приглашения (прежняя ссылка перестаёт работать). Только владелец. Список по умолчанию расшарить нельзя (409). invite_link заполняется, если задан AUTH_TELEGRAM_BOT_USERNAME
// @Tags 			lists
// @Produce 		json
// @Security 		TelegramInitData
// @Param 			id path int true "ID списка"
// @Success 		200 {object} ListDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		403 {object} core_http_response.ErrorResponse "Не владелец"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Список по умолчанию"
// @Router 			/lists/{id}/invite [post]
func (h *ListsHTTPHandler) CreateInvite(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	id, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get list id path value")
		return
	}

	list, err := h.listsService.CreateInvite(ctx, id)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create invite")
		return
	}

	responseHandler.JSONResponse(h.listDTO(list, true), http.StatusOK)
}

// RevokeInvite 	godoc
// @Summary 		Выключить приглашение
// @Description 	Ссылка перестаёт работать, уже вступившие остаются. Только владелец
// @Tags 			lists
// @Security 		TelegramInitData
// @Param 			id path int true "ID списка"
// @Success 		204 "Приглашение выключено"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		403 {object} core_http_response.ErrorResponse "Не владелец"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Router 			/lists/{id}/invite [delete]
func (h *ListsHTTPHandler) RevokeInvite(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	id, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get list id path value")
		return
	}

	if err := h.listsService.RevokeInvite(ctx, id); err != nil {
		responseHandler.ErrorResponse(err, "failed to revoke invite")
		return
	}

	responseHandler.NoContentResponse()
}

type JoinListRequest struct {
	Code string `json:"code" validate:"required,min=8,max=32" example:"pQ3x_Zr8k1LmN0aB"`
}

// JoinList 		godoc
// @Summary 		Вступить в список по приглашению
// @Description 	Код берётся из ссылки t.me/<бот>?startapp=join_<код>. Повторное вступление ничего не меняет
// @Tags 			lists
// @Accept 			json
// @Produce 		json
// @Security 		TelegramInitData
// @Param 			request body JoinListRequest true "Код приглашения"
// @Success 		200 {object} ListDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Приглашение не найдено или выключено"
// @Router 			/lists/join [post]
func (h *ListsHTTPHandler) JoinList(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	var request JoinListRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	list, err := h.listsService.JoinList(ctx, request.Code)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to join list")
		return
	}

	responseHandler.JSONResponse(h.listDTO(list, false), http.StatusOK)
}

// RemoveMember 	godoc
// @Summary 		Исключить участника или выйти из списка
// @Description 	Участник может удалить себя (выйти), владелец — любого участника. Владелец выйти не может (409)
// @Tags 			lists
// @Security 		TelegramInitData
// @Param 			id path int true "ID списка"
// @Param 			user_id path int true "ID участника"
// @Success 		204 "Участник удалён из списка"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		403 {object} core_http_response.ErrorResponse "Не владелец"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Владелец не может выйти"
// @Router 			/lists/{id}/members/{user_id} [delete]
func (h *ListsHTTPHandler) RemoveMember(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	id, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get list id path value")
		return
	}

	userID, err := core_http_request.GetIntPathValue(r, "user_id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get user id path value")
		return
	}

	if err := h.listsService.RemoveMember(ctx, id, userID); err != nil {
		responseHandler.ErrorResponse(err, "failed to remove member")
		return
	}

	responseHandler.NoContentResponse()
}

type SetNotifyChangesRequest struct {
	NotifyChanges *bool `json:"notify_changes" validate:"required" example:"false"`
}

// SetNotifyChanges godoc
// @Summary 		Сообщать об изменениях в списке
// @Description 	Личный переключатель владельца или участника: бот пишет, когда другие добавляют или выполняют задачи. По умолчанию включён
// @Tags 			lists
// @Accept 			json
// @Security 		TelegramInitData
// @Param 			id path int true "ID списка"
// @Param 			request body SetNotifyChangesRequest true "Включить или выключить"
// @Success 		204 "Сохранено"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Router 			/lists/{id}/notifications [put]
func (h *ListsHTTPHandler) SetNotifyChanges(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	id, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get list id path value")
		return
	}

	var request SetNotifyChangesRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	if err := h.listsService.SetNotifyChanges(ctx, id, *request.NotifyChanges); err != nil {
		responseHandler.ErrorResponse(err, "failed to set notify changes")
		return
	}

	responseHandler.NoContentResponse()
}
