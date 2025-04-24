package handler

import (
	"PetStore/internal/model"
	"PetStore/internal/service"
	"PetStore/transport"
	"encoding/json"
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

	if err := h.service.AddPet(r.Context(), &pet); err != nil {
		h.responder.ErrorJSON(w, "failed to add pet", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusCreated, "pet added successfully")
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
	statusValues := r.URL.Query()["status"]
	if len(statusValues) == 0 {
		h.responder.ErrorJSON(w, "status is required", http.StatusBadRequest)
		return
	}

	pets, err := h.service.FindPetsByStatus(r.Context(), statusValues)
	if err != nil {
		h.responder.ErrorJSON(w, "failed to find pets by status", http.StatusBadRequest)
		return
	}

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
	petID := r.URL.Query().Get("id")
	petIDInt, err := strconv.ParseInt(petID, 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
	}
	if petIDInt == 0 {
		h.responder.ErrorJSON(w, "id is empty", http.StatusBadRequest)
		return
	}

	pet, err := h.service.GetPetById(r.Context(), petIDInt)
	if err != nil {
		h.responder.ErrorJSON(w, "failed to find pet by id", http.StatusBadRequest)
	}
	if pet == nil {
		h.responder.ErrorJSON(w, "pet not found", http.StatusNotFound)
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
	petID, err := strconv.ParseInt("petId", 10, 64)
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
	petID, err := strconv.ParseInt("petId", 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid pet ID", http.StatusBadRequest)
		return
	}
	url := r.URL.Query().Get("photoUrls")
	if url == "" {
		h.responder.ErrorJSON(w, "url is required", http.StatusBadRequest)
		return
	}

	if err := h.service.Upload(r.Context(), petID, url); err != nil {
		h.responder.ErrorJSON(w, "failed to delete pet", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, "pet deleted successfully")
}
