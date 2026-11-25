package orders

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The random nonce is safe to store: recovering the bearer token also requires
// the server secret. Domain separation keeps this use independent of sessions.
func recoverTrackingToken(secret []byte, workshopID, orderID uuid.UUID, nonce string) string {
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "tallerflow/tracking/v1/%s/%s/%s", workshopID, orderID, nonce)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (r *PostgresTrackingRepository) ManageLink(ctx context.Context, workshopID, orderID, actorID uuid.UUID, action string) (string, error) {
	if action != "recover" && action != "replace" && action != "revoke" {
		return "", errors.New("invalid tracking action")
	}
	if len(r.secret) == 0 {
		return "", errors.New("tracking requires the configured server secret")
	}
	var token string
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		// Lock the parent, including when no link exists, to serialize concurrent
		// recover/replace/revoke calls for this tenant's order.
		var order struct{ ID uuid.UUID }
		row := tx.Raw("SELECT id FROM orders WHERE id = ? FOR UPDATE", orderID).Scan(&order)
		if row.Error != nil {
			return row.Error
		}
		if row.RowsAffected == 0 {
			return ErrOrderNotFound
		}
		if action == "recover" {
			var link struct {
				Nonce     string
				TokenHash []byte
			}
			row = tx.Raw("SELECT recovery_nonce AS nonce, token_hash FROM order_tracking_links WHERE order_id = ? AND revoked_at IS NULL AND recovery_nonce IS NOT NULL", orderID).Scan(&link)
			if row.Error != nil {
				return row.Error
			}
			if row.RowsAffected > 0 {
				token = recoverTrackingToken(r.secret, workshopID, orderID, link.Nonce)
				if !hmac.Equal(trackingDigest(token), link.TokenHash) {
					return ErrTrackingRecoveryUnavailable
				}
				return nil
			}
		} else {
			if err := tx.Exec("UPDATE order_tracking_links SET revoked_at = ? WHERE order_id = ? AND revoked_at IS NULL", r.now().UTC(), orderID).Error; err != nil {
				return err
			}
		}
		if action != "revoke" {
			nonce, err := security.RandomToken(rand.Reader, trackingTokenBytes)
			if err != nil {
				return err
			}
			token = recoverTrackingToken(r.secret, workshopID, orderID, nonce)
			if err := tx.Exec("INSERT INTO order_tracking_links (workshop_id, order_id, token_hash, recovery_nonce, created_by, created_at) VALUES (?, ?, ?, ?, ?, ?)", workshopID, orderID, trackingDigest(token), nonce, actorID, r.now().UTC()).Error; err != nil {
				return err
			}
		}
		event := map[string]string{"recover": "ORDER_TRACKING_LINK_CREATED", "replace": "ORDER_TRACKING_LINK_REPLACED", "revoke": "ORDER_TRACKING_LINK_REVOKED"}[action]
		return tx.Exec("INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, ?, jsonb_build_object('order_id', ?::text))", workshopID, actorID, event, orderID.String()).Error
	})
	return token, err
}
