package models

import "time"

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	GoldBalance  int64     `json:"gold_balance"`
	Rank         int       `json:"rank"`
	CreatedAt    time.Time `json:"created_at"`
}

type UserProfile struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	GoldBalance int64  `json:"gold_balance"`
	Rank        int    `json:"rank"`
}

func (u *User) ToProfile() UserProfile {
	return UserProfile{
		ID:          u.ID,
		Username:    u.Username,
		GoldBalance: u.GoldBalance,
		Rank:        u.Rank,
	}
}
