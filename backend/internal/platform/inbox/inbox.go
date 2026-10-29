// Package inbox de-duplicates event consumption (doc 06 §8): at-least-once delivery from the outbox becomes
// exactly-once effects when the handler and the inbox record commit in the same transaction.
package inbox

import (
	"context"
	"fmt"
	"time"

	"github.com/NADER-CY/vira/backend/internal/platform/db"
)

// ProcessOnce runs fn for (consumer, eventID) unless it was already processed. It reports whether fn ran.
// fn executes in the same transaction that records the event, so a failure leaves no record and the event
// is processed again on redelivery.
func ProcessOnce(ctx context.Context, pool *db.Pool, consumer, eventID string, fn func(tx db.Tx) error) (bool, error) {
	ran := false
	err := db.InTx(ctx, pool, db.Scope{Audience: db.AudienceSystem}, db.TxOptions{}, func(tx db.Tx) error {
		tag, err := tx.Exec(ctx, `INSERT INTO platform.inbox (consumer, event_id) VALUES ($1, $2)
			ON CONFLICT (consumer, event_id) DO NOTHING`, consumer, eventID)
		if err != nil {
			return fmt.Errorf("inbox: record: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return nil // duplicate delivery
		}
		if err := fn(tx); err != nil {
			return err
		}
		ran = true
		return nil
	})
	return ran, err
}

// Prune deletes inbox records older than `olderThan` (doc 06 §8: 30 days).
func Prune(ctx context.Context, tx db.Tx, olderThan time.Duration) (int64, error) {
	tag, err := tx.Exec(ctx, `DELETE FROM platform.inbox WHERE processed_at < now() - $1::interval`,
		fmt.Sprintf("%d seconds", int64(olderThan.Seconds())))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
