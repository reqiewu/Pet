package handler

import (
	"PetStore/internal/model"
	"PetStore/internal/service"
	"PetStore/transport"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
)

type PetHandler struct {
	service   service.PetService
	responder transport.JSONResponder
}

func NewPetHandler(service service.PetService) *PetHandler {
	return &PetHandler{service: service}
}

// AddPet godoc
// @Summary Add a new pet to the store
// @Description
// @Tags pet
// @Accept  json
// @Produce  json
// @Param pet body models.Pet true "Pet object that needs to be added to the store"
// @Success 201 {object} models.Pet
// @Security ApiKeyAuth
// @Router /pet [post]
func (h *PetHandler) AddPet(w http.ResponseWriter, r *http.Request) {
	var pet model.Pet
	if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Устанавливаем дефолтные значения
	if pet.Status == "" {
		pet.Status = "available" // Дефолтный статус
	}

	if pet.Category == nil {
		pet.Category = &model.Category{} // Дефолтная категория
	}

	if err := h.service.AddPet(r.Context(), &pet); err != nil {
		h.responder.ErrorJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.responder.WriteJSON(w, http.StatusCreated, pet)
}

// UpdatePet godoc
// @Summary Update an existing pet
// @Description
// @Tags pet
// @Accept  json
// @Produce  json
// @Param pet body models.Pet true "Pet object that needs to be added to the store"
// @Success 200 {object} models.Pet
// @Security ApiKeyAuth
// @Router /pet [put]
func (h *PetHandler) UpdatePet(w http.ResponseWriter, r *http.Request) {
	var pet model.Pet
	if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdatePet(r.Context(), &pet); err != nil {
		h.responder.ErrorJSON(w, "failed to update pet", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "pet updated successfully")
}

// FindPetsByStatus godoc
// @Summary Finds Pets by status
// @Description Multiple status values can be provided with comma separated strings
// @Tags pet
// @Accept  json
// @Produce  json
// @Param status query []string true "Status values that need to be considered for filter" collectionFormat(multi)
// @Success 200 {array} models.Pet
// @Security ApiKeyAuth
// @Router /pet/findByStatus [get]
func (h *PetHandler) FindPetsByStatus(w http.ResponseWriter, r *http.Request) {
	var st model.StatusRequest

	// Декодирование тела запроса
	if err := json.NewDecoder(r.Body).Decode(&st); err != nil {
		h.responder.ErrorJSON(w, "неверное тело запроса", http.StatusBadRequest)
		return
	}

	// Проверка наличия статуса
	if len(st.Status) == 0 {
		h.responder.ErrorJSON(w, "статус обязателен", http.StatusBadRequest)
		return
	}

	// Вызов сервисного слоя
	pets, err := h.service.FindPetsByStatus(r.Context(), st.Status)
	if err != nil {
		h.responder.ErrorJSON(w, "не удалось найти питомцев по статусу", http.StatusBadRequest)
		return
	}

	// Отправка успешного ответа
	h.responder.WriteJSON(w, http.StatusOK, pets)
}

// GetPetById godoc
// @Summary Find pet by ID
// @Description Returns a single pet
// @Tags pet
// @Accept  json
// @Produce  json
// @Param petId path int true "ID of pet to return"
// @Success 200 {object} models.Pet
// @Security ApiKeyAuth
// @Router /pet/{petId} [get]
func (h *PetHandler) GetPetById(w http.ResponseWriter, r *http.Request) {
	petID, err := strconv.ParseInt(chi.URLParam(r, "petId"), 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid pet ID", http.StatusBadRequest)
		return
	}

	pet, err := h.service.GetPetById(r.Context(), petID)
	if err != nil {
		h.responder.ErrorJSON(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, pet)
}

// UpdatePetWithForm godoc
// @Summary Updates a pet in the store with form data
// @Description
// @Tags pet
// @Accept  multipart/form-data
// @Produce  json
// @Param petId path int true "ID of pet that needs to be updated"
// @Param name formData string false "Updated name of the pet"
// @Param status formData string false "Updated status of the pet"
// @Success 200 {object} map[string]string
// @Security ApiKeyAuth
// @Router /pet/{petId} [post]
func (h *PetHandler) UpdatePetWithForm(w http.ResponseWriter, r *http.Request) {
	petID, err := strconv.ParseInt(chi.URLParam(r, "petId"), 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid pet ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.responder.ErrorJSON(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := r.FormValue("name")
	status := r.FormValue("status")

	if err := h.service.UpdatePetWithForm(r.Context(), petID, name, status); err != nil {
		h.responder.ErrorJSON(w, "failed to update pet with form", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "pet updated successfully")
}

// DeletePet godoc
// @Summary Deletes a pet
// @Description
// @Tags pet
// @Accept  json
// @Produce  json
// @Param petId path int true "Pet id to delete"
// @Success 200 {object} map[string]string
// @Security ApiKeyAuth
// @Router /pet/{petId} [delete]
func (h *PetHandler) DeletePet(w http.ResponseWriter, r *http.Request) {
	petID, err := strconv.ParseInt("petId", 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid pet ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeletePet(r.Context(), petID); err != nil {
		h.responder.ErrorJSON(w, "failed to delete pet", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "pet deleted successfully")
}

func (h *PetHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	var pet model.Pet
	if err := json.NewDecoder(r.Body).Decode(&pet); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
	}
	if pet.ID <= 0 {
		h.responder.ErrorJSON(w, "invalid id", http.StatusBadRequest)
		return
	}
	if pet.ImageURL == "" {
		h.responder.ErrorJSON(w, "image url is required", http.StatusBadRequest)
		return
	}

	if err := h.service.Upload(r.Context(), pet.ID, pet.ImageURL); err != nil {
		h.responder.ErrorJSON(w, "failed to upload pet", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "image url updated successfully")
}
