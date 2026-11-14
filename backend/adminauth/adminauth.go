// Package adminauth provides two administrative roles:
//
//   - super_admin: authenticates with the ADMIN_TOKEN key from the environment.
//     Has full access and is the only role that can manage admin accounts.
//   - admin: an account created by the super admin, authenticating with a
//     username and password. Has full league/membership access but cannot
//     manage admin accounts.
//
// Both roles send their credential as "Authorization: Bearer <value>". For
// admins the value is an opaque session token issued by POST /api/admin/login;
// only its SHA-256 hash is stored.
package adminauth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	RoleSuperAdmin = "super_admin"
	RoleAdmin      = "admin"

	sessionDuration = 12 * time.Hour
	idleTimeout     = 2 * time.Hour

	ctxRole      = "admin_role"
	ctxUsername  = "admin_username"
	ctxAdminID   = "admin_id"
	ctxPrincipal = "admin_principal"
)

type User struct {
	ID           string     `gorm:"type:uuid;primaryKey" json:"id"`
	Username     string     `gorm:"size:32;not null;uniqueIndex" json:"username"`
	PasswordHash string     `gorm:"size:100;not null" json:"-"`
	Active       bool       `gorm:"not null;default:true" json:"active"`
	CanLeague    bool       `gorm:"not null;default:false" json:"can_league"`
	CanRoaster   bool       `gorm:"not null;default:false" json:"can_roaster"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at"`
}

func (User) TableName() string { return "admin_users" }

type Session struct {
	Hash         string    `gorm:"size:64;primaryKey"`
	AdminID      string    `gorm:"type:uuid;not null;index"`
	ExpiresAt    time.Time `gorm:"not null;index"`
	LastActiveAt time.Time `gorm:"not null"`
	CreatedAt    time.Time
}

func (Session) TableName() string { return "admin_sessions" }

type Attempt struct {
	Key    string `gorm:"size:64;primaryKey"`
	Count  int
	Window time.Time `gorm:"column:window_started_at;index"`
}

func (Attempt) TableName() string { return "admin_login_attempts" }

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&User{}, &Session{}, &Attempt{})
}

// Areas an admin account can be granted. The super admin always has both.
const (
	PermLeague  = "league"  // teams, players, fixtures, results
	PermRoaster = "roaster" // members, dues, cards, donations, fees
)

// Principal is who made an authenticated request.
type Principal struct {
	Role       string
	Username   string
	AdminID    string
	CanLeague  bool
	CanRoaster bool
}

func (p Principal) Can(perm string) bool {
	switch perm {
	case "":
		return true
	case PermLeague:
		return p.CanLeague
	case PermRoaster:
		return p.CanRoaster
	}
	return false
}

// Permissions is the JSON shape sent to the frontend.
func (p Principal) Permissions() map[string]bool {
	return map[string]bool{PermLeague: p.CanLeague, PermRoaster: p.CanRoaster}
}

// Auth checks bearer credentials against the super admin key and admin sessions.
type Auth struct {
	db       *gorm.DB
	superKey [32]byte
	now      func() time.Time
}

func New(db *gorm.DB, superKey string) *Auth {
	return &Auth{db: db, superKey: sha256.Sum256([]byte(superKey)), now: func() time.Time { return time.Now().UTC() }}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func bearer(c *gin.Context) (string, bool) {
	header := c.GetHeader("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	value := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	return value, value != ""
}

var errUnavailable = errors.New("admin sessions unavailable")

// Identify resolves the bearer credential. ok=false means not authenticated.
func (a *Auth) Identify(c *gin.Context) (Principal, bool, error) {
	value, present := bearer(c)
	if !present {
		return Principal{}, false, nil
	}
	got := sha256.Sum256([]byte(value))
	if subtle.ConstantTimeCompare(got[:], a.superKey[:]) == 1 {
		return Principal{Role: RoleSuperAdmin, Username: "Super admin", CanLeague: true, CanRoaster: true}, true, nil
	}
	if len(value) != 64 {
		return Principal{}, false, nil
	}
	if a.db == nil {
		return Principal{}, false, nil
	}
	now := a.now()
	var row struct {
		AdminID    string
		Username   string
		CanLeague  bool
		CanRoaster bool
	}
	err := a.db.WithContext(c.Request.Context()).Table("admin_sessions s").
		Select("s.admin_id, u.username, u.can_league, u.can_roaster").
		Joins("JOIN admin_users u ON u.id = s.admin_id AND u.active = true").
		Where("s.hash = ? AND s.expires_at > ? AND s.last_active_at > ?", hashToken(value), now, now.Add(-idleTimeout)).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Principal{}, false, nil
	}
	if err != nil {
		return Principal{}, false, errUnavailable
	}
	// Sliding idle window; at most one write per minute per session.
	a.db.WithContext(c.Request.Context()).Model(&Session{}).
		Where("hash = ? AND last_active_at < ?", hashToken(value), now.Add(-time.Minute)).
		Update("last_active_at", now)
	return Principal{Role: RoleAdmin, Username: row.Username, AdminID: row.AdminID, CanLeague: row.CanLeague, CanRoaster: row.CanRoaster}, true, nil
}

func (a *Auth) middleware(superOnly bool, perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok, err := a.Identify(c)
		if err != nil {
			c.AbortWithStatusJSON(503, gin.H{"error": "Admin sign-in is temporarily unavailable"})
			return
		}
		if !ok {
			msg := "Admin authentication required"
			if _, present := bearer(c); present {
				msg = "Your admin session is invalid or has expired. Log in again"
			}
			c.AbortWithStatusJSON(401, gin.H{"error": msg})
			return
		}
		if superOnly && p.Role != RoleSuperAdmin {
			c.AbortWithStatusJSON(403, gin.H{"error": "Only the super admin can do this"})
			return
		}
		if !p.Can(perm) {
			area := map[string]string{PermLeague: "League", PermRoaster: "Roaster"}[perm]
			c.AbortWithStatusJSON(403, gin.H{"error": "Your admin account does not have " + area + " permission"})
			return
		}
		c.Set(ctxRole, p.Role)
		c.Set(ctxUsername, p.Username)
		c.Set(ctxAdminID, p.AdminID)
		c.Set(ctxPrincipal, p)
		c.Next()
	}
}

// RequireAdmin allows the super admin and any active admin.
func (a *Auth) RequireAdmin() gin.HandlerFunc { return a.middleware(false, "") }

// RequireLeague allows the super admin and admins with League permission.
func (a *Auth) RequireLeague() gin.HandlerFunc { return a.middleware(false, PermLeague) }

// RequireRoaster allows the super admin and admins with Roaster permission.
func (a *Auth) RequireRoaster() gin.HandlerFunc { return a.middleware(false, PermRoaster) }

// RequireSuperAdmin allows only the super admin key.
func (a *Auth) RequireSuperAdmin() gin.HandlerFunc { return a.middleware(true, "") }

func PrincipalFrom(c *gin.Context) Principal {
	if p, ok := c.Get(ctxPrincipal); ok {
		return p.(Principal)
	}
	return Principal{Role: c.GetString(ctxRole), Username: c.GetString(ctxUsername), AdminID: c.GetString(ctxAdminID)}
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func normalizeUsername(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func usernameValid(s string) bool {
	if len(s) < 3 || len(s) > 32 {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// foldPassword makes admin passwords case-insensitive: they are lowercased
// before hashing and before comparison.
func foldPassword(p string) string { return strings.ToLower(p) }

// Passwords: 8–72 bytes (bcrypt limit), at least one letter and one digit.
func passwordValid(s string) bool {
	if len(s) < 8 || len(s) > 72 {
		return false
	}
	letter, digit := false, false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digit = true
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			letter = true
		}
	}
	return letter && digit
}

func consume(tx *gorm.DB, key string, limit int, window time.Duration, now time.Time) (bool, error) {
	var a Attempt
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&a, "key = ?", key).Error
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

// peer: trust the rightmost X-Forwarded-For only behind a local proxy.
func peer(c *gin.Context) string {
	address := c.RemoteIP()
	ip := net.ParseIP(address)
	if ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
		values := strings.Split(c.GetHeader("X-Forwarded-For"), ",")
		if last := net.ParseIP(strings.TrimSpace(values[len(values)-1])); last != nil {
			address = last.String()
		}
	}
	return hashToken("ip:" + address)
}

// dummyHash keeps failed logins for unknown usernames as slow as real ones.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("not-a-real-password-1"), bcrypt.DefaultCost)
