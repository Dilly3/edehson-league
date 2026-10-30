package membership

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// The first paid dues period, not the receipt date, establishes liability.
// Reconcile only system waivers. Never remove receipts or alter manual waivers.
func reconcileDuesStart(tx *gorm.DB) error {
	var first []struct {
		MemberID string
		Month    string
	}
	if err := tx.Table("club_payments p").Select("p.member_id, MIN(d.month) AS month").Joins("JOIN club_dues d ON d.id=p.due_id AND d.member_id=p.member_id").Where("p.kind='monthly' AND p.void=false AND d.status='paid'").Group("p.member_id").Scan(&first).Error; err != nil {
		return err
	}
	starts := map[string]string{}
	for _, v := range first {
		starts[v.MemberID] = v.Month
	}
	var dues []Due
	if err := tx.Where("status='unpaid' OR auto_waived=true").Order("member_id, month").Find(&dues).Error; err != nil {
		return err
	}
	before, after := map[string][]Due{}, map[string][]Due{}
	for _, d := range dues {
		start := starts[d.MemberID]
		shouldWaive := start != "" && d.Month < start
		if shouldWaive == d.AutoWaived {
			continue
		}
		if d.Status != "unpaid" && !(d.Status == "waived" && d.AutoWaived) {
			continue
		}
		before[d.MemberID] = append(before[d.MemberID], d)
		d.AutoWaived = shouldWaive
		d.Revision++
		if shouldWaive {
			d.Status = "waived"
		} else {
			d.Status = "unpaid"
		}
		if err := tx.Save(&d).Error; err != nil {
			return err
		}
		after[d.MemberID] = append(after[d.MemberID], d)
	}
	for memberID, records := range before {
		old, next := snapshot(records), snapshot(after[memberID])
		digest := sha256.Sum256([]byte(old + next))
		start := starts[memberID]
		if start == "" {
			start = "January (no active monthly payment)"
		}
		audit := CommandRecord{RequestID: uuid.NewString(), Hash: hex.EncodeToString(digest[:]), Type: "dues_start_policy", MemberID: memberID, Note: fmt.Sprintf("Automatic dues waiver reconciliation. First paid dues month: %s. Receipt dates and manual waivers preserved.", start), Before: old, After: next}
		if err := tx.Create(&audit).Error; err != nil {
			return err
		}
	}
	return nil
}
