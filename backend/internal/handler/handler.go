package handler

import (
	"database/sql"
)

type Handler struct {
	db   *sql.DB
	jwt  string
}

func New(db *sql.DB, jwtSecret string) *Handler {
	return &Handler{db: db, jwt: jwtSecret}
}
