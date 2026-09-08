package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrderItem struct {
	MenuItemID int     `json:"menu_item_id"`
	Name       string  `json:"name"`
	Price      float64 `json:"price"`
	Quantity   int     `json:"quantity"`
}

type Order struct {
	ID           int         `json:"id"`
	RestaurantID int         `json:"restaurant_id"`
	Items        []OrderItem `json:"items"`
	TotalPrice   float64     `json:"total_price"`
	Status       string      `json:"status"`
}

func CreateOrder(
	ctx context.Context,
	db *pgxpool.Pool,
	restaurantID int,
	items []OrderItem,
) (Order, error) {

	tx, err := db.Begin(ctx)
	if err != nil {
		return Order{}, err
	}
	defer tx.Rollback(ctx)

	// Проверяем, существует ли ресторан.
	var restaurantExists bool

	err = tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM restaurants
			WHERE id = $1
		)
	`, restaurantID).Scan(&restaurantExists)

	if err != nil {
		return Order{}, err
	}

	if !restaurantExists {
		return Order{}, fmt.Errorf("restaurant not found")
	}

	var totalPrice float64

	for i := range items {
		if items[i].Quantity <= 0 {
			return Order{}, fmt.Errorf("quantity must be greater than zero")
		}

		var (
			name      string
			price     float64
			available bool
		)

		err = tx.QueryRow(ctx, `
			SELECT name, price, available
			FROM menu_items
			WHERE id = $1
			  AND restaurant_id = $2
		`, items[i].MenuItemID, restaurantID).Scan(
			&name,
			&price,
			&available,
		)

		if err == pgx.ErrNoRows {
			return Order{}, fmt.Errorf("menu item not found")
		}

		if err != nil {
			return Order{}, err
		}

		if !available {
			return Order{}, fmt.Errorf("menu item is unavailable")
		}

		items[i].Name = name
		items[i].Price = price

		totalPrice += price * float64(items[i].Quantity)
	}

	var orderID int

	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			restaurant_id,
			status,
			total_price
		)
		VALUES ($1, 'created', $2)
		RETURNING id
	`, restaurantID, totalPrice).Scan(&orderID)

	if err != nil {
		return Order{}, err
	}

	for _, item := range items {
		_, err = tx.Exec(ctx, `
			INSERT INTO order_items (
				order_id,
				menu_item_id,
				name,
				price,
				quantity
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
			orderID,
			item.MenuItemID,
			item.Name,
			item.Price,
			item.Quantity,
		)

		if err != nil {
			return Order{}, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return Order{}, err
	}

	return Order{
		ID:           orderID,
		RestaurantID: restaurantID,
		Items:        items,
		TotalPrice:   totalPrice,
		Status:       "created",
	}, nil
}
func GetOrderByID(
	ctx context.Context,
	db *pgxpool.Pool,
	orderID int,
) (Order, error) {

	var order Order

	err := db.QueryRow(ctx, `
		SELECT id, restaurant_id, total_price, status
		FROM orders
		WHERE id = $1
	`, orderID).Scan(
		&order.ID,
		&order.RestaurantID,
		&order.TotalPrice,
		&order.Status,
	)

	if err == pgx.ErrNoRows {
		return Order{}, fmt.Errorf("order not found")
	}

	if err != nil {
		return Order{}, err
	}

	rows, err := db.Query(ctx, `
		SELECT menu_item_id, name, price, quantity
		FROM order_items
		WHERE order_id = $1
		ORDER BY id
	`, orderID)

	if err != nil {
		return Order{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var item OrderItem

		err := rows.Scan(
			&item.MenuItemID,
			&item.Name,
			&item.Price,
			&item.Quantity,
		)

		if err != nil {
			return Order{}, err
		}

		order.Items = append(order.Items, item)
	}

	if err := rows.Err(); err != nil {
		return Order{}, err
	}

	return order, nil
}
