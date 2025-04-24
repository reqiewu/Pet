package handler

import (
	"PetStore/internal/model"
	"PetStore/internal/service"
	"PetStore/transport"
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	service   service.UserService
	responder transport.JSONResponder
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
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
	h.responder.WriteJSON(w, http.StatusCreated, "user created successfully")
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
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.CreateUsersBatch(r.Context(), users); err != nil {
		h.responder.ErrorJSON(w, "failed to create array", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusCreated, "array created successfully")
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

	if err := h.service.CreateUsersBatch(r.Context(), users); err != nil {
		h.responder.ErrorJSON(w, "failed to create list", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusCreated, "list created successfully")
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
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserName == "" || req.Password == "" {
		h.responder.ErrorJSON(w, "username or password is nil", http.StatusBadRequest)
		return
	}

	token, err := h.service.LoginUser(r.Context(), req.UserName, req.Password)
	if err != nil {
		h.responder.ErrorJSON(w, "username or login is invalid", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, token)
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
	username := r.URL.Query().Get("username")
	if username == "" {
		h.responder.ErrorJSON(w, "username is empty", http.StatusBadRequest)
		return
	}
	user, err := h.service.GetUserByUsername(r.Context(), username)
	if err != nil {
		h.responder.ErrorJSON(w, "failed to get user", http.StatusNotFound)
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
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
	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdateUser(r.Context(), &user); err != nil {
		h.responder.ErrorJSON(w, "failed to update user", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "user updated successfully")
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
