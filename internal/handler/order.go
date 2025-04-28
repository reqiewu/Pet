package handler

import (
	"PetStore/internal/model"
	"PetStore/internal/service"
	"PetStore/transport"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

type StoreHandler struct {
	service   service.StoreService
	responder transport.JSONResponder
}

func NewStoreHandler(service service.StoreService) *StoreHandler {
	return &StoreHandler{service: service}
}

// GetInventory godoc
// @Summary Returns pet inventories by status
// @Description Returns a map of status codes to quantities
// @Tags store
// @Accept  json
// @Produce  json
// @Success 200 {object} map[string]int64
// @Security ApiKeyAuth
// @Router /store/inventory [get]
func (h *StoreHandler) GetInventory(w http.ResponseWriter, r *http.Request) {
	inventory, err := h.service.GetInventory(r.Context())
	if err != nil {
		h.responder.ErrorJSON(w, "error in getting inventory", http.StatusInternalServerError)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, inventory)
}

// PlaceOrder godoc
// @Summary Place an order for a pet
// @Description place an order for a pet
// @Tags store
// @Accept  json
// @Produce  json
// @Param order body model.Order true "order placed for purchasing the pet"
// @Success 200 {object} model.Order
// @Security ApiKeyAuth
// @Router /store/order [post]
func (h *StoreHandler) PlaceOrder(w http.ResponseWriter, r *http.Request) {
	var order model.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		h.responder.ErrorJSON(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.service.PlaceOrder(r.Context(), &order); err != nil {
		if strings.Contains(err.Error(), "invalid") {
			h.responder.ErrorJSON(w, err.Error(), http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "already in an active order") {
			h.responder.ErrorJSON(w, err.Error(), http.StatusConflict)
		} else {
			h.responder.ErrorJSON(w, "failed to place order", http.StatusInternalServerError)
		}
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, order)
}

// GetOrderById godoc
// @Summary Find purchase order by ID
// @Description For valid response try integer IDs with value >= 1 and <= 10. Other values will generated exceptions
// @Tags store
// @Accept  json
// @Produce  json
// @Param orderId path int true "ID of pet that needs to be fetched"
// @Success 200 {object} model.Order
// @Security ApiKeyAuth
// @Router /store/order/{orderId} [get]
func (h *StoreHandler) GetOrderById(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "orderId"), 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid order ID", http.StatusBadRequest)
		return
	}

	order, err := h.service.GetOrderById(r.Context(), orderID)
	if err != nil {
		h.responder.ErrorJSON(w, "failed to get order by id", http.StatusInternalServerError)
		return
	}
	if order == nil {
		h.responder.ErrorJSON(w, "order not found", http.StatusNotFound)
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, order)
}

// DeleteOrder godoc
// @Summary Delete purchase order by ID
// @Description For valid response try integer IDs with positive integer value. Negative or non-integer values will generate API errors
// @Tags store
// @Accept  json
// @Produce  json
// @Param orderId path int true "ID of the order that needs to be deleted"
// @Success 200 {object} map[string]string
// @Security ApiKeyAuth
// @Router /store/order/{orderId} [delete]
func (h *StoreHandler) DeleteOrder(w http.ResponseWriter, r *http.Request) {
	orderID, err := strconv.ParseInt(chi.URLParam(r, "orderId"), 10, 64)
	if err != nil {
		h.responder.ErrorJSON(w, "invalid order ID", http.StatusBadRequest)
		return
	}

	if err := h.service.DeleteOrder(r.Context(), orderID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.responder.ErrorJSON(w, err.Error(), http.StatusNotFound)
		} else {
			h.responder.ErrorJSON(w, "failed to delete order", http.StatusInternalServerError)
		}
		return
	}

	h.responder.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "Order deleted successfully",
	})
}
