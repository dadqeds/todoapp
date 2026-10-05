package users_transport_http

import (
	"net/http"

	core_auth "github.com/dadqeds/todoapp/internal/core/auth"
	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

type GetMeResponse struct {
	UserDTOResponse
	IsAdmin bool `json:"is_admin" example:"false"`
}

// GetMe 			godoc
// @Summary 		Текущий пользователь
// @Description 	Пользователь, от имени которого выполняется запрос. Создаётся автоматически при первом входе через Telegram
// @Tags 			users
// @Produce 		json
// @Security 		TelegramInitData
// @Success 		200 {object} GetMeResponse "Текущий пользователь"
// @Failure 		401 {object} core_http_response.ErrorResponse "Нет или неверные данные Telegram"
// @Router 			/me [get]
func (h *UsersHTTPHandler) GetMe(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	actor, err := core_auth.FromContext(ctx)
	if err != nil {
		responseHandler.ErrorResponse(err, "failed to get current user")
		return
	}

	responseHandler.JSONResponse(GetMeResponse{
		UserDTOResponse: userDTOFromDomain(actor.User),
		IsAdmin:         actor.IsAdmin,
	}, http.StatusOK)
}
