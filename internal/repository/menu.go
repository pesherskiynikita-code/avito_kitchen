package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MenuItem struct {
	ID           int     `json:"id"`
	RestaurantID int     `json:"restaurant_id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Price        float64 `json:"price"`
	Available    bool    `json:"available"`
}

func GetMenuByRestaurant(
	ctx context.Context,
	db *pgxpool.Pool,
	restaurantID int,
) ([]MenuItem, error) {
	rows, err := db.Query(ctx, `
		SELECT id, restaurant_id, name, description, price, available
		FROM menu_items
		WHERE restaurant_id = $1
		ORDER BY id
	`, restaurantID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MenuItem

	for rows.Next() {
		var item MenuItem

		err := rows.Scan(
			&item.ID,
			&item.RestaurantID,
			&item.Name,
			&item.Description,
			&item.Price,
			&item.Available,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}
