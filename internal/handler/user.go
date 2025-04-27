package handler

import (
	"PetStore/internal/model"
	"PetStore/internal/service"
	"PetStore/transport"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strings"
)

type UserHandler struct {
	service   service.UserService
	responder transport.JSONResponder
}

func NewUserHandler(service service.UserService, responder transport.JSONResponder) *UserHandler {
	return &UserHandler{service: service, responder: responder}
}

// CreateUser godoc
// @Summary Create user
// @Description This can only be done by the logged in user.
// @Tags user
// @Accept  json
// @Produce  json
// @Param user body models.User true "Created user object"
// @Success 200 {object} models.User
// @Router /user [post]
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateUser(r.Context(), &user); err != nil {
		h.responder.ErrorJSON(w, "failed to create user", http.StatusBadRequest)
		return
	}
	response := map[string]interface{}{
		"message": "user created successfully",
		"id":      user.ID,
	}
	h.responder.WriteJSON(w, http.StatusCreated, response)
}

// CreateUsersWithArray godoc
// @Summary Creates list of users with given input array
// @Tags user
// @Accept  json
// @Produce  json
// @Param users body []models.User true "List of user objects"
// @Success 200 {object} map[string]string
// @Router /user/createWithArray [post]
func (h *UserHandler) CreateUsersWithArray(w http.ResponseWriter, r *http.Request) {
	var users []*model.User
	if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
		h.responder.ErrorJSON(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if len(users) == 0 {
		h.responder.ErrorJSON(w, "empty users array", http.StatusBadRequest)
		return
	}

	ids, err := h.service.CreateUsersBatch(r.Context(), users)
	if err != nil {
		h.responder.ErrorJSON(w, "failed to create users: "+err.Error(), http.StatusBadRequest)
		return
	}
	response := struct {
		Message string  `json:"message"`
		UserIDs []int64 `json:"userIds"`
	}{
		Message: fmt.Sprintf("Successfully created %d users", len(ids)),
		UserIDs: ids,
	}

	h.responder.WriteJSON(w, http.StatusCreated, response)
}

// CreateUsersWithList godoc
// @Summary Creates list of users with given input list
// @Tags user
// @Accept  json
// @Produce  json
// @Param users body []models.User true "List of user objects"
// @Success 200 {object} map[string]string
// @Router /user/createWithList [post]
func (h *UserHandler) CreateUsersWithList(w http.ResponseWriter, r *http.Request) {
	var users []*model.User
	if err := json.NewDecoder(r.Body).Decode(&users); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	ids, err := h.service.CreateUsersBatch(r.Context(), users)
	if err != nil {
		h.responder.ErrorJSON(w, "failed to create users: "+err.Error(), http.StatusBadRequest)
		return
	}
	response := struct {
		Message string  `json:"message"`
		UserIDs []int64 `json:"userIds"`
	}{
		Message: fmt.Sprintf("Successfully created %d users", len(ids)),
		UserIDs: ids,
	}

	h.responder.WriteJSON(w, http.StatusCreated, response)
}

// LoginUser godoc
// @Summary Logs user into the system
// @Tags user
// @Accept  json
// @Produce  json
// @Param username query string true "The username for login"
// @Param password query string true "The password for login in clear text"
// @Success 200 {object} map[string]string
// @Router /user/login [get]
func (h *UserHandler) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req model.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.responder.ErrorJSON(w, "invalid request format", http.StatusBadRequest)
		return
	}

	if req.UserName == "" || req.Password == "" {
		h.responder.ErrorJSON(w, "username and password are required", http.StatusBadRequest)
		return
	}

	token, err := h.service.LoginUser(r.Context(), req.UserName, req.Password)
	if err != nil {
		// Различаем типы ошибок для клиента
		errorMsg := "authentication failed"
		if strings.Contains(err.Error(), "user not found") {
			errorMsg = "user not found"
		} else if strings.Contains(err.Error(), "invalid password") {
			errorMsg = "invalid password"
		}

		h.responder.ErrorJSON(w, errorMsg, http.StatusUnauthorized)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, model.LoginResponse{Token: token})
}

// LogoutUser godoc
// @Summary Logs out current loggedin user session
// @Tags user
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]string
// @Router /user/logout [get]
func (h *UserHandler) LogoutUser(w http.ResponseWriter, r *http.Request) {
	if err := h.service.LogoutUser(); err != nil {
		h.responder.ErrorJSON(w, "failed to logout", http.StatusBadRequest)
	}
	h.responder.WriteJSON(w, http.StatusOK, "user logout successfully")
}

// GetUserByName godoc
// @Summary Get user by username
// @Tags user
// @Accept  json
// @Produce  json
// @Param username path string true "The name that needs to be fetched. Use user1 for testing."
// @Success 200 {object} models.User
// @Router /user/{username} [get]
func (h *UserHandler) GetUserByName(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username string `json:"username"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if request.Username == "" {
		h.responder.ErrorJSON(w, "username is empty", http.StatusBadRequest)
		return
	}
	user, err := h.service.GetUserByUsername(r.Context(), request.Username)
	if err != nil {
		h.responder.ErrorJSON(w, "user not found", http.StatusNotFound)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, user)
}

// UpdateUser godoc
// @Summary Updated user
// @Description This can only be done by the logged in user.
// @Tags user
// @Accept  json
// @Produce  json
// @Param username path string true "name that need to be updated"
// @Param user body models.User true "Updated user object"
// @Success 200 {object} models.User
// @Router /user/{username} [put]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")

	var updateData model.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем, что хотя бы одно поле передано для обновления
	if updateData.FirstName == nil && updateData.LastName == nil &&
		updateData.Email == nil && updateData.Password == nil &&
		updateData.Phone == nil {
		h.responder.ErrorJSON(w, "no valid fields provided for update", http.StatusBadRequest)
		return
	}

	// Преобразуем в map для репозитория
	updateMap := make(map[string]interface{})
	if updateData.FirstName != nil {
		updateMap["first_name"] = *updateData.FirstName
	}
	if updateData.LastName != nil {
		updateMap["last_name"] = *updateData.LastName
	}
	if updateData.Email != nil {
		updateMap["email"] = *updateData.Email
	}
	if updateData.Password != nil {
		updateMap["password"] = *updateData.Password
	}
	if updateData.Phone != nil {
		updateMap["phone"] = *updateData.Phone
	}

	if err := h.service.UpdateUser(r.Context(), username, updateMap); err != nil {
		h.responder.ErrorJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "user updated successfully",
	})
}

// DeleteUser godoc
// @Summary Delete user
// @Description This can only be done by the logged in user.
// @Tags user
// @Accept  json
// @Produce  json
// @Param username path string true "The name that needs to be deleted"
// @Success 200 {object} map[string]string
// @Router /user/{username} [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.service.DeleteUser(r.Context(), user.UserName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "user deleted successfully")
}
