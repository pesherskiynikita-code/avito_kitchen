package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type OrderRequest struct {
	OrderID      int `json:"order_id"`
	RestaurantID int `json:"restaurant_id"`
}

type OrderResponse struct {
	OrderID int    `json:"order_id"`
	Status  string `json:"status"`
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var order OrderRequest

	err := json.NewDecoder(r.Body).Decode(&order)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	fmt.Printf(
		"Restaurant received order: order_id=%d restaurant_id=%d\n",
		order.OrderID,
		order.RestaurantID,
	)

	response := OrderResponse{
		OrderID: order.OrderID,
		Status:  "accepted",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(response)
}

func main() {
	http.HandleFunc("/orders", orderHandler)

	fmt.Println("Restaurant service started on :8081")

	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}
