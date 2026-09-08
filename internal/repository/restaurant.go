package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Restaurant struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
}

func GetAllRestaurants(ctx context.Context, db *pgxpool.Pool) ([]Restaurant, error) {
	rows, err := db.Query(ctx, `
		SELECT id, name, address, status
		FROM restaurants
		ORDER BY id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var restaurants []Restaurant

	for rows.Next() {
		var restaurant Restaurant

		err := rows.Scan(
			&restaurant.ID,
			&restaurant.Name,
			&restaurant.Address,
			&restaurant.Status,
		)
		if err != nil {
			return nil, err
		}

		restaurants = append(restaurants, restaurant)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return restaurants, nil
}
