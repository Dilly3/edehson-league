package membership

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// An imported count represents a group, not separately dated events. Split it
// only when an administrator edits an individual card; preserve settlement totals.
func editIndividualCard(tx *gorm.DB, in Input, member Member, log *CommandRecord) error {
	if _, err := uuid.Parse(in.RecordID); err != nil {
		return invalid("Select a card")
	}
	if !validDate(in.OccurredOn) || in.OccurredOn < member.JoinedOn || in.OccurredOn > today() {
		return invalid("Choose a card issue date on or after joining and not in the future")
	}
	var original Card
	if err := tx.First(&original, "id = ? AND member_id = ?", in.RecordID, member.ID).Error; err != nil {
		return err
	}
	if original.Void {
		return conflict("A void card cannot be edited")
	}
	if in.ExpectedOccurredOn == nil || original.OccurredOn != *in.ExpectedOccurredOn || in.ExpectedCount != original.Count {
		return conflict("This card record changed. Refresh before saving")
	}
	if in.CardIndex < 0 || in.CardIndex >= original.Count {
		return invalid("Select an individual card")
	}
	var allocations []Allocation
	if err := tx.Joins("JOIN club_payments p ON p.id = club_allocations.payment_id").Where("club_allocations.card_id = ? AND p.void = false", original.ID).Order("club_allocations.id").Find(&allocations).Error; err != nil {
		return err
	}
	paid := 0
	for _, a := range allocations {
		paid += a.Count
	}
	if paid > original.Count {
		return conflict("Card allocations exceed the received count; review this record")
	}
	// Match the UI's deterministic assignment of group settlements to card rows.
	position := 0
	for _, a := range allocations {
		if in.CardIndex >= position && in.CardIndex < position+a.Count {
			var p Payment
			if err := tx.First(&p, "id = ?", a.PaymentID).Error; err != nil {
				return err
			}
			if p.PaidOn != "" && in.OccurredOn > p.PaidOn {
				return invalid("Card issue date cannot be after its payment date")
			}
		}
		position += a.Count
	}
	log.Before = snapshot(map[string]interface{}{"card": original, "allocations": allocations})
	cards := []Card{original}
	if original.Count > 1 {
		cards[0].Count = 1
		if err := tx.Save(&cards[0]).Error; err != nil {
			return err
		}
		for i := 1; i < original.Count; i++ {
			c := original
			c.ID = uuid.NewString()
			c.Count = 1
			if err := tx.Create(&c).Error; err != nil {
				return err
			}
			cards = append(cards, c)
		}
		position = 0
		for _, a := range allocations {
			count := a.Count
			a.CardID = cards[position].ID
			a.Count = 1
			if err := tx.Save(&a).Error; err != nil {
				return err
			}
			position++
			for i := 1; i < count; i++ {
				next := Allocation{PaymentID: a.PaymentID, CardID: cards[position].ID, Count: 1}
				if err := tx.Create(&next).Error; err != nil {
					return err
				}
				position++
			}
		}
	}
	cards[in.CardIndex].OccurredOn = in.OccurredOn
	if err := tx.Save(&cards[in.CardIndex]).Error; err != nil {
		return err
	}
	// Keep original notes, rates and payment amounts; the reason is in the audit.
	var afterAllocations []Allocation
	ids := make([]string, len(cards))
	for i, c := range cards {
		ids[i] = c.ID
	}
	if err := tx.Where("card_id IN ?", ids).Find(&afterAllocations).Error; err != nil {
		return err
	}
	log.After = snapshot(map[string]interface{}{"cards": cards, "allocations": afterAllocations})
	return nil
}
