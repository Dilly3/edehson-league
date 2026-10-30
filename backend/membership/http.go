package membership

import (
	"database/sql"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
)

type RegisterData struct {
	AnnualFees  []AnnualFee  `json:"annual_fees"`
	Members     []Member     `json:"members"`
	Fees        []Fee        `json:"fees"`
	Dues        []Due        `json:"dues"`
	Cards       []Card       `json:"cards"`
	Payments    []Payment    `json:"payments"`
	Allocations []Allocation `json:"allocations"`
}

// api is the existing /api group. Pass the existing admin middleware.
func Register(api *gin.RouterGroup, db *gorm.DB, admin gin.HandlerFunc) {
	g := api.Group("/membership")
	g.Use(func(c *gin.Context) { c.Header("Cache-Control", "no-store"); c.Next() })
	g.GET("", func(c *gin.Context) {
		year, err := strconv.Atoi(c.Query("year"))
		if err != nil || year < 2026 || year > 2200 {
			c.JSON(400, gin.H{"error": "Choose a year from 2026 to 2200"})
			return
		}
		if err := EnsureDues(db.WithContext(c.Request.Context())); err != nil {
			log.Printf("Accrue membership dues: %v", err)
			c.JSON(500, gin.H{"error": "Could not update monthly dues"})
			return
		}
		out := RegisterData{AnnualFees: []AnnualFee{}, Members: []Member{}, Fees: []Fee{}, Dues: []Due{}, Cards: []Card{}, Payments: []Payment{}, Allocations: []Allocation{}}
		err = db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			for _, q := range []struct {
				out   interface{}
				query *gorm.DB
			}{
				{&out.Members, tx.Order("LOWER(name), id")},
				{&out.AnnualFees, tx.Order("year DESC")},
				{&out.Fees, tx.Order("effective_from DESC, kind")},
				{&out.Dues, tx.Where("month >= ? AND month <= ?", strconv.Itoa(year)+"-01", strconv.Itoa(year)+"-12")},
				{&out.Cards, tx.Order("occurred_on, created_at, id")},
				{&out.Payments, tx.Order("paid_on, created_at, id")},
				{&out.Allocations, tx},
			} {
				if err := q.query.Find(q.out).Error; err != nil {
					return err
				}
			}
			for i := range out.AnnualFees {
				var n int64
				y := out.AnnualFees[i].Year
				if err := tx.Model(&Due{}).Where("month >= ? AND month <= ?", strconv.Itoa(y)+"-01", strconv.Itoa(y)+"-12").Count(&n).Error; err != nil {
					return err
				}
				out.AnnualFees[i].Locked = n > 0
			}
			return nil
		}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
		if err != nil {
			log.Printf("Read membership register: %v", err)
			c.JSON(500, gin.H{"error": "Could not load membership register"})
			return
		}
		c.JSON(200, gin.H{"data": out})
	})
	g.GET("/members/:memberID/history", func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("memberID"))
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid member ID"})
			return
		}
		var m Member
		if err := db.WithContext(c.Request.Context()).First(&m, "id = ?", id.String()).Error; err != nil {
			status, msg := explain(err)
			c.JSON(status, gin.H{"error": msg})
			return
		}
		history := []CommandRecord{}
		if err := db.WithContext(c.Request.Context()).Where("member_id = ?", id.String()).Order("created_at DESC, id").Find(&history).Error; err != nil {
			c.JSON(500, gin.H{"error": "Could not load history"})
			return
		}
		dues := []Due{}
		if err := db.WithContext(c.Request.Context()).Where("member_id = ?", id.String()).Find(&dues).Error; err != nil {
			c.JSON(500, gin.H{"error": "Could not load dues history"})
			return
		}
		for _, d := range dues {
			history = append(history, CommandRecord{Base: Base{ID: d.ID, CreatedAt: d.CreatedAt}, Type: "issue_dues", MemberID: m.ID, After: snapshot(map[string]interface{}{"month": d.Month, "amount": d.Amount})})
		}
		sort.SliceStable(history, func(i, j int) bool { return history[i].CreatedAt.After(history[j].CreatedAt) })
		c.JSON(200, gin.H{"data": history})
	})
	g.POST("/commands", admin, func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		var in Input
		if err := decoder.Decode(&in); err != nil {
			c.JSON(400, gin.H{"error": "Provide one valid membership command"})
			return
		}
		var extra interface{}
		if err := decoder.Decode(&extra); err != io.EOF {
			c.JSON(400, gin.H{"error": "Provide exactly one JSON object"})
			return
		}
		if err := execute(db.WithContext(c.Request.Context()), in); err != nil {
			status, msg := explain(err)
			if status == 500 {
				log.Printf("Membership command: %v", err)
			}
			c.JSON(status, gin.H{"error": msg})
			return
		}
		c.Status(http.StatusNoContent)
	})
}
