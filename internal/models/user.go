package models

import "time"

type User struct {
	ID        int64     `db:"id" json:"id"`
	Email     string    `db:"email" json:"email"`
	Password  string    `db:"password_hash" json:"-"`
	Role      Role      `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
