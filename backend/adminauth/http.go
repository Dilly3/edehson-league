package adminauth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func decode(c *gin.Context, out interface{}) bool {
	if !strings.HasPrefix(c.ContentType(), "application/json") {
		c.JSON(400, gin.H{"error": "Send JSON content"})
		return false
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 2048)
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

func unavailable(c *gin.Context) {
	c.AbortWithStatusJSON(503, gin.H{"error": "Admin service is temporarily unavailable"})
}

const usernameRule = "Usernames are 3–32 characters: lowercase letters, numbers, dots, dashes or underscores"
const permissionRule = "Give the admin League permission, Roaster permission, or both"

var errNoPermission = errors.New("admin needs at least one permission")

const passwordRule = "Passwords need 8–72 characters with at least one letter and one number"

// Register mounts the admin session and account-management routes on /api.
func (a *Auth) Register(api *gin.RouterGroup) {
	db := a.db

	api.POST("/admin/login", func(c *gin.Context) {
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if !decode(c, &in) {
			return
		}
		username := normalizeUsername(in.Username)
		if username == "" || in.Password == "" || len(in.Password) > 72 {
			c.JSON(400, gin.H{"error": "Enter your username and password"})
			return
		}
		token, err := newToken()
		if err != nil {
			unavailable(c)
			return
		}
		status := 200
		var user User
		err = db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			now := a.now()
			for _, b := range []struct {
				key    string
				limit  int
				window time.Duration
			}{{"global", 60, time.Minute}, {peer(c), 10, 15 * time.Minute}, {hashToken("user:" + username), 10, 15 * time.Minute}} {
				ok, err := consume(tx, b.key, b.limit, b.window, now)
				if err != nil {
					return err
				}
				if !ok {
					status = 429
					return nil
				}
			}
			err := tx.Where("username = ? AND active = true", username).Take(&user).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(foldPassword(in.Password)))
				_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(in.Password)) // match the two-compare cost of a wrong password
				status = 401
				return nil
			}
			if err != nil {
				return err
			}
			if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(foldPassword(in.Password))) != nil {
				// Accounts created before passwords became case-insensitive were hashed as typed.
				if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
					status = 401
					return nil
				}
				if hash, err := bcrypt.GenerateFromPassword([]byte(foldPassword(in.Password)), bcrypt.DefaultCost); err == nil {
					if err := tx.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
						return err
					}
				}
			}
			if err := tx.Where("expires_at <= ? OR last_active_at <= ?", now, now.Add(-idleTimeout)).Delete(&Session{}).Error; err != nil {
				return err
			}
			if err := tx.Where("window_started_at < ?", now.Add(-24*time.Hour)).Delete(&Attempt{}).Error; err != nil {
				return err
			}
			if err := tx.Model(&user).Update("last_login_at", now).Error; err != nil {
				return err
			}
			return tx.Create(&Session{Hash: hashToken(token), AdminID: user.ID, ExpiresAt: now.Add(sessionDuration), LastActiveAt: now}).Error
		})
		if err != nil {
			unavailable(c)
			return
		}
		switch status {
		case 429:
			c.Header("Retry-After", "900")
			c.JSON(429, gin.H{"error": "Too many sign-in attempts. Try again in 15 minutes."})
		case 401:
			c.JSON(401, gin.H{"error": "Incorrect username or password"})
		default:
			p := Principal{Role: RoleAdmin, Username: user.Username, CanLeague: user.CanLeague, CanRoaster: user.CanRoaster}
			c.JSON(200, gin.H{"data": gin.H{"token": token, "role": RoleAdmin, "username": user.Username, "permissions": p.Permissions()}})
		}
	})

	api.GET("/admin/session", a.RequireAdmin(), func(c *gin.Context) {
		p := PrincipalFrom(c)
		c.JSON(200, gin.H{"data": gin.H{"authenticated": true, "role": p.Role, "username": p.Username, "permissions": p.Permissions()}})
	})

	api.POST("/admin/logout", a.RequireAdmin(), func(c *gin.Context) {
		if p := PrincipalFrom(c); p.Role == RoleAdmin {
			token, _ := bearer(c)
			if err := db.WithContext(c.Request.Context()).Where("hash = ?", hashToken(token)).Delete(&Session{}).Error; err != nil {
				unavailable(c)
				return
			}
		}
		c.Status(204)
	})

	users := api.Group("/admin/users", a.RequireSuperAdmin())

	users.GET("", func(c *gin.Context) {
		var list []User
		if err := db.WithContext(c.Request.Context()).Order("username").Find(&list).Error; err != nil {
			unavailable(c)
			return
		}
		c.JSON(200, gin.H{"data": list})
	})

	users.POST("", func(c *gin.Context) {
		var in struct {
			Username   string `json:"username"`
			Password   string `json:"password"`
			CanLeague  *bool  `json:"can_league"`
			CanRoaster *bool  `json:"can_roaster"`
		}
		if !decode(c, &in) {
			return
		}
		league, roaster := in.CanLeague == nil || *in.CanLeague, in.CanRoaster == nil || *in.CanRoaster
		if !league && !roaster {
			c.JSON(400, gin.H{"error": permissionRule})
			return
		}
		username := normalizeUsername(in.Username)
		if !usernameValid(username) {
			c.JSON(400, gin.H{"error": usernameRule})
			return
		}
		if !passwordValid(in.Password) {
			c.JSON(400, gin.H{"error": passwordRule})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(foldPassword(in.Password)), bcrypt.DefaultCost)
		if err != nil {
			unavailable(c)
			return
		}
		user := User{ID: uuid.NewString(), Username: username, PasswordHash: string(hash), Active: true, CanLeague: league, CanRoaster: roaster}
		created, err := insertUser(db.WithContext(c.Request.Context()), &user)
		if err != nil {
			unavailable(c)
			return
		}
		if !created {
			c.JSON(409, gin.H{"error": "That username is already taken"})
			return
		}
		c.JSON(201, gin.H{"data": user})
	})

	// PATCH accepts any of: active (bool), password (string).
	users.PATCH("/:id", func(c *gin.Context) {
		var in struct {
			Active     *bool   `json:"active"`
			Password   *string `json:"password"`
			CanLeague  *bool   `json:"can_league"`
			CanRoaster *bool   `json:"can_roaster"`
		}
		if !decode(c, &in) {
			return
		}
		if in.Active == nil && in.Password == nil && in.CanLeague == nil && in.CanRoaster == nil {
			c.JSON(400, gin.H{"error": "Nothing to update"})
			return
		}
		updates := map[string]interface{}{}
		if in.Password != nil {
			if !passwordValid(*in.Password) {
				c.JSON(400, gin.H{"error": passwordRule})
				return
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(foldPassword(*in.Password)), bcrypt.DefaultCost)
			if err != nil {
				unavailable(c)
				return
			}
			updates["password_hash"] = string(hash)
		}
		if in.Active != nil {
			updates["active"] = *in.Active
		}
		if in.CanLeague != nil {
			updates["can_league"] = *in.CanLeague
		}
		if in.CanRoaster != nil {
			updates["can_roaster"] = *in.CanRoaster
		}
		id := c.Param("id")
		if _, err := uuid.Parse(id); err != nil {
			c.JSON(404, gin.H{"error": "Admin not found"})
			return
		}
		var user User
		err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			res := tx.Model(&User{}).Where("id = ?", id).Updates(updates)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
			var check User
			if err := tx.First(&check, "id = ?", id).Error; err != nil {
				return err
			}
			if !check.CanLeague && !check.CanRoaster {
				return errNoPermission
			}
			// A new password or deactivation signs the admin out everywhere.
			if in.Password != nil || (in.Active != nil && !*in.Active) {
				if err := tx.Where("admin_id = ?", id).Delete(&Session{}).Error; err != nil {
					return err
				}
			}
			return tx.First(&user, "id = ?", id).Error
		})
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(404, gin.H{"error": "Admin not found"})
			return
		}
		if errors.Is(err, errNoPermission) {
			c.JSON(400, gin.H{"error": permissionRule})
			return
		}
		if err != nil {
			unavailable(c)
			return
		}
		c.JSON(200, gin.H{"data": user})
	})

	users.DELETE("/:id", func(c *gin.Context) {
		id := c.Param("id")
		if _, err := uuid.Parse(id); err != nil {
			c.JSON(404, gin.H{"error": "Admin not found"})
			return
		}
		found := true
		err := db.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("admin_id = ?", id).Delete(&Session{}).Error; err != nil {
				return err
			}
			res := tx.Where("id = ?", id).Delete(&User{})
			found = res.RowsAffected == 1
			return res.Error
		})
		if err != nil {
			unavailable(c)
			return
		}
		if !found {
			c.JSON(404, gin.H{"error": "Admin not found"})
			return
		}
		c.Status(204)
	})
}

// insertUser creates an admin, reporting false if the username is taken.
// Select("*") is essential: without it GORM omits false booleans that have a
// column default, so can_league/can_roaster=false would be stored as true.
func insertUser(db *gorm.DB, user *User) (bool, error) {
	res := db.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "username"}}, DoNothing: true}).
		Select("*").Create(user)
	return res.RowsAffected == 1, res.Error
}
