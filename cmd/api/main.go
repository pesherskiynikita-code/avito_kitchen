package main

import (
	"avito-kitchen/internal/database"
	"avito-kitchen/internal/repository"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func main() {
	db, err := database.Connect()
	if err != nil {
		fmt.Println("Database connection error:", err)
		return
	}
	defer db.Close()
	fmt.Println("Database connected")

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/restaurants", restaurantsDBHandler(db))
	http.HandleFunc("/restaurants/", restaurantsRouter(db))
	http.HandleFunc("/orders", ordersHandler(db))
	http.HandleFunc("/orders/", orderRouter(db))
	fmt.Println("Server started on http://localhost:8080")

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := HealthResponse{
		Status: "ok",
	}

	json.NewEncoder(w).Encode(response)
}

func restaurantHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		pathParts := strings.Split(r.URL.Path, "/")
		idStr := pathParts[len(pathParts)-1]

		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var restaurant repository.Restaurant

		err = db.QueryRow(r.Context(), `
			SELECT id, name, address, status
			FROM restaurants
			WHERE id = $1
		`, id).Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Address,
			&restaurant.Status,
		)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(restaurant)
	}
}

func restaurantsDBHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		restaurants, err := repository.GetAllRestaurants(r.Context(), db)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(restaurants)
	}
}

func menuDBHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		pathParts := strings.Split(r.URL.Path, "/")
		idStr := pathParts[len(pathParts)-2]

		restaurantID, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		items, err := repository.GetMenuByRestaurant(
			r.Context(),
			db,
			restaurantID,
		)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}

func restaurantsRouter(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		pathParts := strings.Split(path, "/")

		if len(pathParts) == 3 {
			restaurantHandler(db)(w, r)
		} else if len(pathParts) == 4 && pathParts[3] == "menu" {
			menuDBHandler(db)(w, r)
		}
	}
}

func ordersHandler(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			RestaurantID int `json:"restaurant_id"`
			Items        []struct {
				MenuItemID int `json:"menu_item_id"`
				Quantity   int `json:"quantity"`
			} `json:"items"`
		}

		err := json.NewDecoder(r.Body).Decode(&request)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if request.RestaurantID <= 0 || len(request.Items) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		items := make([]repository.OrderItem, 0, len(request.Items))

		for _, item := range request.Items {
			items = append(items, repository.OrderItem{
				MenuItemID: item.MenuItemID,
				Quantity:   item.Quantity,
			})
		}

		order, err := repository.CreateOrder(
			r.Context(),
			db,
			request.RestaurantID,
			items,
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		restaurantRequest := struct {
			OrderID      int `json:"order_id"`
			RestaurantID int `json:"restaurant_id"`
		}{
			OrderID:      order.ID,
			RestaurantID: order.RestaurantID,
		}

		requestBody, err := json.Marshal(restaurantRequest)
		if err != nil {
			http.Error(w, "failed to create restaurant request", http.StatusInternalServerError)
			return
		}

		restaurantServiceURL := os.Getenv("RESTAURANT_SERVICE_URL")
		if restaurantServiceURL == "" {
			restaurantServiceURL = "http://localhost:8081"
		}

		resp, err := http.Post(
			restaurantServiceURL+"/orders",
			"application/json",
			bytes.NewBuffer(requestBody),
		)
		if err != nil {
			http.Error(w, "restaurant service unavailable", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			http.Error(w, fmt.Sprintf("restaurant service error: %s", string(body)),
				http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		json.NewEncoder(w).Encode(order)
	}
}

func orderRouter(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		pathParts := strings.Split(r.URL.Path, "/")
		idStr := pathParts[len(pathParts)-1]

		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		order, err := repository.GetOrderByID(
			r.Context(),
			db,
			id,
		)

		if err != nil {
			if err.Error() == "order not found" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(order)
	}
}
