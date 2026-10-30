package vieweraccess

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const cookieName = "greatminds_viewer"
const sessionDuration = 24 * time.Hour
const idleTimeout = 2 * time.Hour

type Setting struct {
	ID        int    `gorm:"primaryKey"`
	PINHash   string `gorm:"size:100;not null" json:"-"`
	Version   int64  `gorm:"not null"`
	UpdatedAt time.Time
}

func (Setting) TableName() string { return "viewer_access_settings" }

type Session struct {
	Hash         string    `gorm:"size:64;primaryKey"`
	Version      int64     `gorm:"not null"`
	ExpiresAt    time.Time `gorm:"not null;index"`
	LastActiveAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"`
}

func (Session) TableName() string { return "viewer_access_sessions" }

type Attempt struct {
	Key    string `gorm:"size:64;primaryKey"`
	Count  int
	Window time.Time `gorm:"column:window_started_at;index"`
}

func (Attempt) TableName() string { return "viewer_access_attempts" }

type Audit struct {
	ID              string `gorm:"type:uuid;primaryKey"`
	CreatedAt       time.Time
	PreviousVersion int64
	Version         int64
}

func (Audit) TableName() string { return "viewer_access_audits" }

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&Setting{}, &Session{}, &Attempt{}, &Audit{}); err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		if err := lock(tx); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&Setting{}).Where("id = 1").Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return nil
		}
		hash, err := bcrypt.GenerateFromPassword([]byte("GM4492"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		return tx.Create(&Setting{ID: 1, PINHash: string(hash), Version: 1}).Error
	})
}
func lock(tx *gorm.DB) error { return tx.Exec("SELECT pg_advisory_xact_lock(7231942)").Error }
func pinValid(pin string) bool {
	if len(pin) != 6 {
		return false
	}
	letter, digit := false, false
	for _, c := range pin {
		switch {
		case c >= '0' && c <= '9':
			digit = true
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
			letter = true
		default:
			return false
		}
	}
	return letter && digit
}
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
func adminValid(c *gin.Context, key string) bool {
	if key == "" {
		return false
	}
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	expected := sha256.Sum256([]byte(key))
	got := sha256.Sum256([]byte(strings.TrimPrefix(header, "Bearer ")))
	return subtle.ConstantTimeCompare(expected[:], got[:]) == 1
}
func authError(c *gin.Context) {
	c.AbortWithStatusJSON(401, gin.H{"error": "Enter the viewer PIN to continue", "code": "viewer_auth_required"})
}
func serverError(c *gin.Context) {
	c.AbortWithStatusJSON(503, gin.H{"error": "Access service is temporarily unavailable"})
}
func sessionValid(db *gorm.DB, c *gin.Context) (bool, error) {
	token, err := c.Cookie(cookieName)
	if err != nil || len(token) != 64 {
		return false, nil
	}
	if db == nil {
		return false, errors.New("database unavailable")
	}
	var count int64
	err = db.WithContext(c.Request.Context()).Model(&Session{}).Joins("JOIN viewer_access_settings s ON s.id = 1 AND s.version = viewer_access_sessions.version").Where("viewer_access_sessions.hash = ? AND expires_at > ?", hashToken(token), time.Now().UTC()).Where("last_active_at > ?", time.Now().UTC().Add(-idleTimeout)).Count(&count).Error
	return count == 1, err
}

// Applied before every /api route, including direct-origin requests.
func Middleware(db *gorm.DB, adminKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.HasPrefix(c.Request.URL.Path, "/api/") && c.Request.URL.Path != "/api" {
			c.Next()
			return
		}
		c.Header("Cache-Control", "no-store")
		switch c.Request.URL.Path {
		case "/api/health", "/api/ready", "/api/admin/session", "/api/viewer/login", "/api/viewer/session", "/api/viewer/logout":
			c.Next()
			return
		}
		if adminValid(c, adminKey) {
			c.Next()
			return
		}
		ok, err := sessionValid(db, c)
		if err != nil {
			serverError(c)
			return
		}
		if !ok {
			authError(c)
			return
		}
		c.Next()
	}
}
func decode(c *gin.Context, out interface{}) bool {
	if !strings.HasPrefix(c.ContentType(), "application/json") {
		c.JSON(400, gin.H{"error": "Send JSON content"})
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		c.JSON(400, gin.H{"error": "Provide valid input"})
		return false
	}
	if d.Decode(new(interface{})) != io.EOF {
		c.JSON(400, gin.H{"error": "Provide one JSON object"})
		return false
	}
	return true
}
func setCookie(c *gin.Context, value string, maxAge int) {
	host, _, err := net.SplitHostPort(c.Request.Host)
	if err != nil {
		host = c.Request.Host
	}
	secure := host != "localhost" && host != "127.0.0.1" && host != "::1"
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(cookieName, value, maxAge, "/api", "", secure, true)
}

// Caddy discards untrusted incoming X-Forwarded-For values and supplies the
// immediate upstream IP. Use the rightmost value, never Gin's trust-all default.
func peer(c *gin.Context) string {
	address := c.RemoteIP()
	ip := net.ParseIP(address)
	if ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
		values := strings.Split(c.GetHeader("X-Forwarded-For"), ",")
		if last := net.ParseIP(strings.TrimSpace(values[len(values)-1])); last != nil {
			address = last.String()
		}
	}
	return hashToken(address)
}
func consume(tx *gorm.DB, key string, limit int, window time.Duration, now time.Time) (bool, error) {
	var a Attempt
	err := tx.First(&a, "key = ?", key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		a = Attempt{Key: key, Window: now}
	} else if err != nil {
		return false, err
	}
	if now.Sub(a.Window) >= window {
		a.Window = now
		a.Count = 0
	}
	if a.Count >= limit {
		return false, nil
	}
	a.Count++
	return true, tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&a).Error
}
func Register(api *gin.RouterGroup, db *gorm.DB, admin gin.HandlerFunc) {
	api.POST("/viewer/login", func(c *gin.Context) {
		var in struct {
			PIN string `json:"pin"`
		}
		if !decode(c, &in) {
			return
		}
		if !pinValid(in.PIN) {
			c.JSON(400, gin.H{"error": "Enter exactly six characters with at least one letter and one number"})
			return
		}
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			serverError(c)
			return
		}
		token := hex.EncodeToString(tokenBytes)
		status := 200
		err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := lock(tx); err != nil {
				return err
			}
			now := time.Now().UTC()
			for _, bucket := range []struct {
				key    string
				limit  int
				window time.Duration
			}{{"global", 60, time.Minute}, {peer(c), 10, 15 * time.Minute}} {
				ok, err := consume(tx, bucket.key, bucket.limit, bucket.window, now)
				if err != nil {
					return err
				}
				if !ok {
					status = 429
					return nil
				}
			}
			var setting Setting
			if err := tx.First(&setting, 1).Error; err != nil {
				return err
			}
			if bcrypt.CompareHashAndPassword([]byte(setting.PINHash), []byte(in.PIN)) != nil {
				status = 401
				return nil
			}
			if err := tx.Where("expires_at <= ?", now).Delete(&Session{}).Error; err != nil {
				return err
			}
			if err := tx.Where("window_started_at < ?", now.Add(-24*time.Hour)).Delete(&Attempt{}).Error; err != nil {
				return err
			}
			if old, err := c.Cookie(cookieName); err == nil {
				if err := tx.Where("hash = ?", hashToken(old)).Delete(&Session{}).Error; err != nil {
					return err
				}
			}
			return tx.Create(&Session{Hash: hashToken(token), Version: setting.Version, ExpiresAt: now.Add(sessionDuration), LastActiveAt: now}).Error
		})
		if err != nil {
			serverError(c)
			return
		}
		if status == 429 {
			c.Header("Retry-After", "900")
			c.JSON(429, gin.H{"error": "Too many PIN attempts. Try again in 15 minutes."})
			return
		}
		if status == 401 {
			c.JSON(401, gin.H{"error": "Incorrect viewer PIN"})
			return
		}
		setCookie(c, token, int(sessionDuration.Seconds()))
		c.JSON(200, gin.H{"data": gin.H{"authenticated": true, "idle_remaining_seconds": int(idleTimeout.Seconds())}})
	})
	api.GET("/viewer/session", func(c *gin.Context) {
		ok, err := sessionValid(db, c)
		if err != nil {
			serverError(c)
			return
		}
		if !ok {
			authError(c)
			return
		}
		token, _ := c.Cookie(cookieName)
		var session Session
		if err := db.WithContext(c.Request.Context()).First(&session, "hash = ?", hashToken(token)).Error; err != nil {
			serverError(c)
			return
		}
		remaining := time.Until(session.LastActiveAt.Add(idleTimeout))
		if absolute := time.Until(session.ExpiresAt); absolute < remaining {
			remaining = absolute
		}
		if remaining <= 0 {
			authError(c)
			return
		}
		c.JSON(200, gin.H{"data": gin.H{"authenticated": true, "idle_remaining_seconds": int(remaining.Seconds())}})
	})
	api.POST("/viewer/activity", func(c *gin.Context) {
		var in struct{}
		if !decode(c, &in) {
			return
		}
		token, err := c.Cookie(cookieName)
		if err != nil {
			authError(c)
			return
		}
		now := time.Now().UTC()
		result := db.WithContext(c.Request.Context()).Model(&Session{}).Where("hash = ? AND expires_at > ? AND last_active_at > ? AND version = (SELECT version FROM viewer_access_settings WHERE id = 1)", hashToken(token), now, now.Add(-idleTimeout)).Update("last_active_at", now)
		if result.Error != nil {
			serverError(c)
			return
		}
		if result.RowsAffected != 1 {
			authError(c)
			return
		}
		c.Status(204)
	})
	api.POST("/viewer/logout", func(c *gin.Context) {
		if token, err := c.Cookie(cookieName); err == nil {
			if err := db.WithContext(c.Request.Context()).Where("hash = ?", hashToken(token)).Delete(&Session{}).Error; err != nil {
				serverError(c)
				return
			}
		}
		setCookie(c, "", -1)
		c.Status(204)
	})
	api.GET("/admin/viewer-pin", admin, func(c *gin.Context) {
		var setting Setting
		if err := db.WithContext(c.Request.Context()).First(&setting, 1).Error; err != nil {
			serverError(c)
			return
		}
		c.JSON(200, gin.H{"data": gin.H{"version": setting.Version}})
	})
	api.PUT("/admin/viewer-pin", admin, func(c *gin.Context) {
		var in struct {
			PIN     string `json:"pin"`
			Version *int64 `json:"version"`
		}
		if !decode(c, &in) {
			return
		}
		if !pinValid(in.PIN) || in.Version == nil {
			c.JSON(400, gin.H{"error": "Enter exactly six characters with at least one letter and one number and refresh the access settings"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(in.PIN), bcrypt.DefaultCost)
		if err != nil {
			serverError(c)
			return
		}
		stale := false
		err = db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := lock(tx); err != nil {
				return err
			}
			var setting Setting
			if err := tx.First(&setting, 1).Error; err != nil {
				return err
			}
			if setting.Version != *in.Version {
				stale = true
				return nil
			}
			old := setting.Version
			setting.PINHash = string(hash)
			setting.Version++
			if err := tx.Save(&setting).Error; err != nil {
				return err
			}
			if err := tx.Where("1 = 1").Delete(&Session{}).Error; err != nil {
				return err
			}
			return tx.Create(&Audit{ID: uuid.NewString(), PreviousVersion: old, Version: setting.Version}).Error
		})
		if err != nil {
			serverError(c)
			return
		}
		if stale {
			c.JSON(409, gin.H{"error": "The PIN changed. Refresh before saving"})
			return
		}
		c.Status(204)
	})
}
