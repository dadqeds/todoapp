package users_transport_http

import (
	"net/http"

	core_logger "github.com/dadqeds/todoapp/internal/core/logger"
	core_http_request "github.com/dadqeds/todoapp/internal/core/transport/http/request"
	core_http_response "github.com/dadqeds/todoapp/internal/core/transport/http/response"
)

// DeleteUser 		godoc
// @Summary 		Удаление пользователя
// @Description 	Удаление существующего пользователя по его ID
// @Tags 			users
// @Param			id path int true "ID удаляемого пользователя"
// @Success 		204	"Успешное удаление пользователя"
// @Failure 		400 {object} core_http_response.ErrorResponse "Bad request"
// @Failure 		404 {object} core_http_response.ErrorResponse "Not found"
// @Failure 		500 {object} core_http_response.ErrorResponse "Internal server error"
// @Router 			/users/{id} [delete]
func (h *UsersHTTPHandler) DeleteUser(rw http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := core_logger.FromContext(ctx)
	responseHandler := core_http_response.NewHTTPResponseHandler(log, rw)

	userId, err := core_http_request.GetIntPathValue(r, "id")
	if err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to get userID path value",
		)
		return
	}

	if err := h.usersService.DeleteUser(ctx, userId); err != nil {
		responseHandler.ErrorResponse(
			err,
			"failed to delete user",
		)
		return
	}

	responseHandler.NoContentResponse()
}
