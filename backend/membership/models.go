package membership

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type Base struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (b *Base) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return nil
}

type Member struct {
	Base
	Name     string `gorm:"size:80;not null" json:"name"`
	JoinedOn string `gorm:"size:10;not null" json:"joined_on"`
	Active   bool   `gorm:"not null" json:"active"`
	Revision int    `gorm:"not null" json:"revision"`
}

func (Member) TableName() string { return "club_members" }

type Fee struct {
	Base
	Kind          string `gorm:"size:12;not null;uniqueIndex:club_fee_date" json:"kind"`
	EffectiveFrom string `gorm:"size:10;not null;uniqueIndex:club_fee_date" json:"effective_from"`
	Amount        int64  `gorm:"not null;check:club_fee_positive,amount > 0" json:"amount"`
}

func (Fee) TableName() string { return "club_fees" }

type AnnualFee struct {
	Base
	Year     int   `gorm:"not null;uniqueIndex" json:"year"`
	Amount   int64 `gorm:"not null;check:club_annual_fee_positive,amount > 0" json:"amount"`
	Revision int   `gorm:"not null" json:"revision"`
	Locked   bool  `gorm:"-" json:"locked"`
}

func (AnnualFee) TableName() string { return "club_annual_fees" }

type Due struct {
	AutoWaived bool `gorm:"not null;default:false" json:"auto_waived"`
	Base
	MemberID string `gorm:"type:uuid;not null;uniqueIndex:club_due_month" json:"member_id"`
	Member   Member `gorm:"foreignKey:MemberID;constraint:OnDelete:RESTRICT" json:"-"`
	Month    string `gorm:"size:7;not null;uniqueIndex:club_due_month" json:"month"`
	Amount   int64  `gorm:"not null;check:club_due_positive,amount > 0" json:"amount"`
	Status   string `gorm:"size:10;not null;check:club_due_status,status IN ('unpaid','paid','waived')" json:"status"`
	Revision int    `gorm:"not null" json:"revision"`
}

func (Due) TableName() string { return "club_dues" }

type Card struct {
	Base
	MemberID   string `gorm:"type:uuid;not null;index" json:"member_id"`
	Member     Member `gorm:"foreignKey:MemberID;constraint:OnDelete:RESTRICT" json:"-"`
	Color      string `gorm:"size:6;not null;check:club_card_color,color IN ('yellow','red')" json:"color"`
	Count      int    `gorm:"not null;check:club_card_count,count > 0" json:"count"`
	UnitAmount int64  `gorm:"not null;check:club_card_fee,unit_amount > 0" json:"unit_amount"`
	OccurredOn string `gorm:"size:10;not null" json:"occurred_on"`
	Note       string `gorm:"size:300" json:"note"`
	Void       bool   `gorm:"not null" json:"void"`
}

func (Card) TableName() string { return "club_cards" }

type Payment struct {
	Base
	MemberID string  `gorm:"type:uuid;not null;index" json:"member_id"`
	Member   Member  `gorm:"foreignKey:MemberID;constraint:OnDelete:RESTRICT" json:"-"`
	Kind     string  `gorm:"size:12;not null" json:"kind"`
	DueID    *string `gorm:"type:uuid" json:"due_id"`
	Due      *Due    `gorm:"foreignKey:DueID;constraint:OnDelete:RESTRICT" json:"-"`
	Amount   int64   `gorm:"not null;check:club_payment_positive,amount > 0" json:"amount"`
	PaidOn   string  `gorm:"size:10;not null" json:"paid_on"`
	Note     string  `gorm:"size:300" json:"note"`
	Void     bool    `gorm:"not null" json:"void"`
}

func (Payment) TableName() string { return "club_payments" }

type Allocation struct {
	Base
	PaymentID string  `gorm:"type:uuid;not null;index" json:"payment_id"`
	Payment   Payment `gorm:"foreignKey:PaymentID;constraint:OnDelete:RESTRICT" json:"-"`
	CardID    string  `gorm:"type:uuid;not null;index" json:"card_id"`
	Card      Card    `gorm:"foreignKey:CardID;constraint:OnDelete:RESTRICT" json:"-"`
	Count     int     `gorm:"not null;check:club_allocation_count,count > 0" json:"count"`
}

func (Allocation) TableName() string { return "club_allocations" }

// Every write has an immutable command entry. Reusing a request ID with the
// same command is a safe retry; different contents are rejected.
type CommandRecord struct {
	Base
	RequestID string `gorm:"type:uuid;not null;uniqueIndex" json:"request_id"`
	Hash      string `gorm:"size:64;not null" json:"-"`
	Type      string `gorm:"size:20;not null" json:"type"`
	MemberID  string `gorm:"size:36;index" json:"member_id"`
	Note      string `gorm:"size:300" json:"note"`
	Before    string `gorm:"type:text" json:"before"`
	After     string `gorm:"type:text" json:"after"`
}

func (CommandRecord) TableName() string { return "club_commands" }

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Member{}, &Fee{}, &AnnualFee{}, &Due{}, &Card{}, &Payment{}, &Allocation{}, &CommandRecord{}); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(7231941)").Error; err != nil {
			return err
		}
		for kind, amount := range map[string]int64{"monthly": 100000, "yellow": 50000, "red": 100000} {
			var n int64
			if err := tx.Model(&Fee{}).Where("kind = ?", kind).Count(&n).Error; err != nil {
				return err
			}
			if n == 0 {
				if err := tx.Create(&Fee{Kind: kind, EffectiveFrom: "1900-01-01", Amount: amount}).Error; err != nil {
					return err
				}
			}
		}
		var annual AnnualFee
		if err := tx.Where("year = ?", 2026).Attrs(AnnualFee{Year: 2026, Amount: 100000}).FirstOrCreate(&annual).Error; err != nil {
			return err
		}
		if err := ensureDuesTx(tx, today()); err != nil {
			return err
		}
		return defaultHistoricalCardDates(tx)
	})
}
