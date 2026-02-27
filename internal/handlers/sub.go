package handlers

import (
	"encoding/json"
	"fmt"
	"naivepanel/internal/database"
	"naivepanel/internal/models"
	"net/http"
	"strings"
	"time"
)

// SubHandler handles the public subscription API for NaiveUI clients
type SubHandler struct {
	DB *database.DB
}

func (h *SubHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userAgent := r.Header.Get("User-Agent")
	if !strings.HasPrefix(userAgent, "NaiveUI/") {
		http.Error(w, "Forbidden: Invalid User-Agent", http.StatusForbidden)
		return
	}

	settings, _ := h.DB.GetSettings()
	subPathPrefix := "/sub/"
	if settings != nil && settings.SubPath != "" {
		subPathPrefix = "/" + settings.SubPath + "/"
	}

	path := strings.TrimPrefix(r.URL.Path, subPathPrefix)
	token := strings.TrimSpace(path)
	if token == "" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	user, err := h.DB.GetUserBySubToken(token)
	if err != nil || user == nil {
		http.Error(w, "Subscription not found", http.StatusNotFound)
		return
	}

	if !user.Enabled {
		// Return 200 OK with empty config if disabled
		h.respondEmpty(w, "Account disabled")
		return
	}

	// 1. Check expiration
	if user.ExpiresAt != nil && user.ExpiresAt.Before(time.Now()) {
		h.respondEmpty(w, "Subscription expired")
		return
	}

	// 2. Auto HWID Reset if interval is set
	if user.HWIDResetInterval > 0 {
		var lastReset time.Time
		if user.LastHWIDReset != nil {
			lastReset = *user.LastHWIDReset
		} else {
			lastReset = user.CreatedAt
		}
		
		intervalDuration := time.Duration(user.HWIDResetInterval) * 24 * time.Hour
		if time.Since(lastReset) > intervalDuration {
			h.DB.ResetHWIDs(user.ID)
			h.DB.UpdateHWIDResetTime(user.ID)
		}
	}

	// 3. HWID Management
	hwid := r.Header.Get("X-HWID")
	if user.HWIDLimit > 0 && hwid != "" {
		hwids, _ := h.DB.GetHWIDs(user.ID)
		
		isKnown := false
		for _, h := range hwids {
			if h.HWID == hwid {
				isKnown = true
				break
			}
		}

		if !isKnown {
			if len(hwids) >= user.HWIDLimit {
				h.respondEmpty(w, "HWID limit reached")
				return
			}
			// Register new HWID
			err := h.DB.RegisterHWID(user.ID, hwid)
			if err != nil {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}
	}

	// 4. Build config response
	settings, _ = h.DB.GetSettings()
	domain := settings.Domain
	if domain == "" {
		domain = "your-server-ip"
	}
	port := settings.Port
	if port == 0 {
		port = 443
	}

	var exp int64 = 0
	if user.ExpiresAt != nil {
		exp = user.ExpiresAt.Unix()
	}

	subData := models.SubscriptionData{
		Version: 1,
		Info: &models.SubscriptionInfo{
			UserTag:           user.Username,
			ExpiresAt:         exp,
			TrafficLimitBytes: user.TrafficLimit,
			TrafficUsedBytes:  user.TrafficUp + user.TrafficDown,
			Message:           "",
		},
		Profiles: []models.SubscriptionProfile{
			{
				Name:           fmt.Sprintf("Panel Node (%s)", domain),
				Server:         domain,
				Port:           port,
				Username:       user.Username,
				Password:       user.Password,
				Protocol:       "https",
				ListenProtocol: "socks",
				ListenPort:     1080,
				Concurrency:    1,
				ExtraHeaders:   "", // No extra headers by default
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(subData)
}

func (h *SubHandler) respondEmpty(w http.ResponseWriter, message string) {
	subData := models.SubscriptionData{
		Version:  1,
		Info:     &models.SubscriptionInfo{Message: message},
		Profiles: []models.SubscriptionProfile{},
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(subData)
}
