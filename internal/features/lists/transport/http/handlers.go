package lists_transport_http

import (
	"net/http"

	"github.com/dadqeds/todoapp/internal/core/domain"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
	core_http_types "github.com/dadqeds/todoapp/internal/core/transport/http/types"
)

// GetLists 		godoc
// @Summary 		Мои списки
// @Description 	Свои и общие списки текущего пользователя со счётчиками, ролью и участниками. Список по умолчанию идёт первым и создаётся автоматически. Администратор может передать user_id
// @Tags 			lists
// @Produce 		json
// @Security 		TelegramInitData
// @Param 			user_id query int false "Чьи списки показать (только для администратора)"
// @Success 		200 {array} ListSummaryDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/lists [get]
func (h *ListsHTTPHandler) GetLists(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	userID, err := core_http_request.GetIntQueryParam(r, "user_id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get 'user_id' query param")
		return
	}

	lists, err := h.listsService.GetLists(ctx, userID)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get lists")
		return
	}

	response := make([]ListSummaryDTOResponse, len(lists))
	for i, l := range lists {
		response[i] = h.listSummaryDTO(l)
	}

	responseHandler.JSONResponse(response, http.StatusOK)
}

type CreateListRequest struct {
	Title string `json:"title" validate:"required,min=1,max=50" example:"Дом"`
	Color string `json:"color" validate:"required"              example:"green" enums:"green,violet,coral,blue,pink,amber"`
}

// CreateList 		godoc
// @Summary 		Создать список
// @Tags 			lists
// @Accept 			json
// @Produce 		json
// @Security 		TelegramInitData
// @Param 			request body CreateListRequest true "Новый список"
// @Success 		201 {object} ListDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/lists [post]
func (h *ListsHTTPHandler) CreateList(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	var request CreateListRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	list, err := h.listsService.CreateList(ctx, domain.NewListUninitialized(request.Title, request.Color, 0))
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to create list")
		return
	}

	responseHandler.JSONResponse(h.listDTO(list, true), http.StatusCreated)
}

type PatchListRequest struct {
	Title   core_http_types.Nullable[string] `json:"title"   swaggertype:"string" example:"Дача"`
	Color   core_http_types.Nullable[string] `json:"color"   swaggertype:"string" example:"amber"`
	Version *int                             `json:"version" example:"1"`
}

// PatchList 		godoc
// @Summary 		Изменить список
// @Description 	Название и цвет, только владелец (участнику — 403). Необязательное поле 'version' защищает от одновременного изменения (409)
// @Tags 			lists
// @Accept 			json
// @Produce 		json
// @Security 		TelegramInitData
// @Param 			id path int true "ID списка"
// @Param 			request body PatchListRequest true "Изменения"
// @Success 		200 {object} ListDTOResponse
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Conflict"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/lists/{id} [patch]
func (h *ListsHTTPHandler) PatchList(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	id, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get list id path value")
		return
	}

	var request PatchListRequest
	if err := core_http_request.DecodeAndValidateRequest(r, &request); err != nil {
		responseHandler.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	list, err := h.listsService.PatchList(ctx, id, domain.ListPatch{
		Title:           request.Title.ToDomain(),
		Color:           request.Color.ToDomain(),
		ExpectedVersion: request.Version,
	})
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to patch list")
		return
	}

	responseHandler.JSONResponse(h.listDTO(list, true), http.StatusOK)
}

// DeleteList 		godoc
// @Summary 		Удалить список
// @Description 	Удаляет список вместе со всеми его задачами, только владелец (участнику — 403). Список по умолчанию удалить нельзя (409)
// @Tags 			lists
// @Security 		TelegramInitData
// @Param 			id path int true "ID списка"
// @Success 		204 "Список удалён"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		409 {object} core_http_response.ErrorResponse "Список по умолчанию"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/lists/{id} [delete]
func (h *ListsHTTPHandler) DeleteList(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	responseHandler := core_http_response.NewHTTPResponseHandler(core_logger.FromContext(ctx), rw)

	id, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get list id path value")
		return
	}

	if err := h.listsService.DeleteList(ctx, id); err != nil {
		responseHandler.ErrorResponse(err, "failed to delete list")
		return
	}

	responseHandler.NoContentResponse()
}
