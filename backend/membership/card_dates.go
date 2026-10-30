package membership

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const historicalCardDate = "2026-01-01"

// The user chose this opening date for undated historical card records.
// Preserve source-date provenance in audit history; leave known dates intact.
func defaultHistoricalCardDates(tx *gorm.DB) error {
	var members []Member
	if err := tx.Where("id IN (SELECT member_id FROM club_cards WHERE occurred_on = '' AND void = false UNION SELECT member_id FROM club_payments WHERE paid_on = '' AND kind IN ('yellow','red') AND void = false)").Find(&members).Error; err != nil {
		return err
	}
	for _, m := range members {
		var cards []Card
		var payments []Payment
		if err := tx.Where("member_id = ? AND occurred_on = '' AND void = false", m.ID).Find(&cards).Error; err != nil {
			return err
		}
		if err := tx.Where("member_id = ? AND paid_on = '' AND kind IN ('yellow','red') AND void = false", m.ID).Find(&payments).Error; err != nil {
			return err
		}
		before := snapshot(map[string]interface{}{"cards": cards, "payments": payments})
		date := historicalCardDate
		if m.JoinedOn > date {
			date = m.JoinedOn
		}
		for i := range cards {
			cards[i].OccurredOn = date
			if err := tx.Save(&cards[i]).Error; err != nil {
				return err
			}
		}
		for i := range payments {
			paidOn := date
			var linked []Card
			if err := tx.Joins("JOIN club_allocations a ON a.card_id = club_cards.id").Where("a.payment_id = ?", payments[i].ID).Find(&linked).Error; err != nil {
				return err
			}
			for _, c := range linked {
				if c.OccurredOn > paidOn {
					paidOn = c.OccurredOn
				}
			}
			payments[i].PaidOn = paidOn
			if err := tx.Save(&payments[i]).Error; err != nil {
				return err
			}
		}
		after := snapshot(map[string]interface{}{"cards": cards, "payments": payments})
		hash := sha256.Sum256([]byte(before + after))
		audit := CommandRecord{RequestID: uuid.NewString(), Hash: hex.EncodeToString(hash[:]), Type: "default_card_dates", MemberID: m.ID, Note: "Admin chose 1 January 2026 for undated historical cards/payments. These are default dates, not source-recorded dates. Joining/known issue dates take precedence if later.", Before: before, After: after}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}
	}
	return nil
}
