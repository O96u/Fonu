package notify

import (
	"context"
	"database/sql"
)

const deliveryHistoryLimit = 1000

func insertDelivery(ctx context.Context, db *sql.DB, row Delivery) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "INSERT INTO notify_delivery_history(at,event,channel,state,detail) VALUES(?,?,?,?,?)", row.At, row.Event, row.Channel, row.State, row.Detail); err != nil {
		return err
	}
	// Trim by monotonic insertion ID, independent of clock adjustments.
	if _, err = tx.ExecContext(ctx, "DELETE FROM notify_delivery_history WHERE id <= (SELECT id FROM notify_delivery_history ORDER BY id DESC LIMIT 1 OFFSET ?)", deliveryHistoryLimit); err != nil {
		return err
	}
	return tx.Commit()
}

func listDeliveries(ctx context.Context, db *sql.DB) ([]Delivery, error) {
	rows, err := db.QueryContext(ctx, "SELECT at,event,channel,state,detail FROM notify_delivery_history ORDER BY id DESC LIMIT ?", deliveryHistoryLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]Delivery, 0)
	for rows.Next() {
		var row Delivery
		if err := rows.Scan(&row.At, &row.Event, &row.Channel, &row.State, &row.Detail); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
