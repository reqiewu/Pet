package handler

import (
	"PetStore/internal/model"
	"PetStore/internal/service"
	"PetStore/transport"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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
// @Param pet body model.Pet true "Pet object that needs to be added to the store"
// @Success 201 {object} model.Pet
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

	// Инициализируем пустой массив для photoUrls, если он nil
	if pet.ImageURL == nil {
		pet.ImageURL = make([]string, 0)
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
// @Param pet body model.Pet true "Pet object that needs to be added to the store"
// @Success 200 {object} model.Pet
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
// @Success 200 {array} model.Pet
// @Security ApiKeyAuth
// @Router /pet/findByStatus [get]
func (h *PetHandler) FindPetsByStatus(w http.ResponseWriter, r *http.Request) {
	statusStr := r.URL.Query().Get("status")
	fmt.Printf("Received status parameter: %s\n", statusStr)

	if statusStr == "" {
		h.responder.ErrorJSON(w, "status is required", http.StatusBadRequest)
		return
	}

	// Разбиваем строку на массив статусов
	statuses := strings.Split(statusStr, ",")
	for i := range statuses {
		statuses[i] = strings.TrimSpace(statuses[i])
	}
	fmt.Printf("Parsed statuses: %v\n", statuses)

	// Проверяем каждый статус
	for _, status := range statuses {
		if status != "available" && status != "pending" && status != "sold" {
			fmt.Printf("Invalid status found: %s\n", status)
			h.responder.ErrorJSON(w, fmt.Sprintf("invalid status: %s", status), http.StatusBadRequest)
			return
		}
	}

	// Выводим данные из таблицы pets для отладки
	if err := h.service.DebugPets(r.Context()); err != nil {
		fmt.Printf("Debug error: %v\n", err)
	}

	// Преобразуем массив статусов в строку для сервиса
	statusString := strings.Join(statuses, ",")
	fmt.Printf("Sending to service: %s\n", statusString)

	pets, err := h.service.FindPetsByStatus(r.Context(), statusString)
	if err != nil {
		fmt.Printf("Service error: %v\n", err)
		h.responder.ErrorJSON(w, "failed to find pets by status", http.StatusBadRequest)
		return
	}

	fmt.Printf("Found %d pets\n", len(pets))
	h.responder.WriteJSON(w, http.StatusOK, pets)
}

// GetPetById godoc
// @Summary Find pet by ID
// @Description Returns a single pet
// @Tags pet
// @Accept  json
// @Produce  json
// @Param petId path int true "ID of pet to return"
// @Success 200 {object} model.Pet
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

	// Определяем тип контента
	contentType := r.Header.Get("Content-Type")

	var name, status string

	if strings.Contains(contentType, "application/json") {
		// Обработка JSON запроса
		var updateData struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
			h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
			return
		}
		name = updateData.Name
		status = updateData.Status
	} else {
		// Обработка form-data
		if err := r.ParseForm(); err != nil {
			h.responder.ErrorJSON(w, "failed to parse form data", http.StatusBadRequest)
			return
		}
		name = r.FormValue("name")
		status = r.FormValue("status")
	}

	if name == "" && status == "" {
		h.responder.ErrorJSON(w, "at least one field (name or status) must be provided", http.StatusBadRequest)
		return
	}

	if err := h.service.UpdatePetWithForm(r.Context(), petID, name, status); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.responder.ErrorJSON(w, err.Error(), http.StatusNotFound)
		} else {
			h.responder.ErrorJSON(w, "failed to update pet: "+err.Error(), http.StatusBadRequest)
		}
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Pet updated successfully",
	})
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
	petID, err := strconv.ParseInt(chi.URLParam(r, "petId"), 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid pet ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeletePet(r.Context(), petID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.responder.ErrorJSON(w, err.Error(), http.StatusNotFound)
		} else {
			h.responder.ErrorJSON(w, "failed to delete pet", http.StatusBadRequest)
		}
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Pet deleted successfully",
	})
}

// UploadImage godoc
// @Summary Uploads an image
// @Description
// @Tags pet
// @Accept  json
// @Produce  json
// @Param petId path int true "ID of pet to update"
// @Param image body object true "Image to upload"
// @Success 200 {object} map[string]string
// @Security ApiKeyAuth
// @Router /pet/{petId}/uploadImage [post]
func (h *PetHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	petID, err := strconv.ParseInt(chi.URLParam(r, "petId"), 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid pet ID", http.StatusBadRequest)
		return
	}

	var updateData struct {
		PhotoUrls []string `json:"photoUrls"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(updateData.PhotoUrls) == 0 {
		h.responder.ErrorJSON(w, "at least one photo URL must be provided", http.StatusBadRequest)
		return
	}

	// Обновляем все URL изображений
	if err := h.service.Upload(r.Context(), petID, updateData.PhotoUrls); err != nil {
		h.responder.ErrorJSON(w, "failed to upload images", http.StatusBadRequest)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Images uploaded successfully",
	})
}
