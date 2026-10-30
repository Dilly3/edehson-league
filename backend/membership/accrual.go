package membership

import (
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const RegisterStart = "2026-01"

// EnsureDues catches up missing monthly charges using the server's WAT date.
// Reading the register may materialize charges, but cannot select their dates,
// amounts or members. Automatic pre-membership waivers are reconciled with payments.
func EnsureDues(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(7231941)").Error; err != nil {
			return err
		}
		return ensureDuesTx(tx, today())
	})
}

func ensureDuesTx(tx *gorm.DB, asOf string) error {
	if !validDate(asOf) {
		return fmt.Errorf("invalid accrual date")
	}
	if asOf[:7] < RegisterStart {
		return nil
	}
	var members []Member
	if err := tx.Where("active = ? AND joined_on <= ?", true, asOf).Find(&members).Error; err != nil {
		return err
	}
	var fees []AnnualFee
	if err := tx.Where("year >= 2026 AND year <= ?", asOf[:4]).Find(&fees).Error; err != nil {
		return err
	}
	dues := []Due{}
	for _, m := range members {
		for _, fee := range fees {
			start := fmt.Sprintf("%04d-01", fee.Year)
			end := fmt.Sprintf("%04d-12", fee.Year)
			// No payment means January liability in the membership year.
			if m.JoinedOn[:4] > fmt.Sprintf("%04d", fee.Year) {
				continue
			}
			for month := start; month <= end; month = nextMonth(month)[:7] {
				dues = append(dues, Due{MemberID: m.ID, Month: month, Amount: fee.Amount, Status: "unpaid"})
			}
		}
	}
	if len(dues) > 0 {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "member_id"}, {Name: "month"}}, DoNothing: true}).CreateInBatches(&dues, 100).Error; err != nil {
			return err
		}
	}
	return reconcileDuesStart(tx)
}
