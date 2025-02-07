package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

type Entry struct {
	User  string `json:"user"`
	Score int    `json:"score"`
}

type StorageWrapper struct {
	db *sql.DB
}

func NewStorage() (StorageWrapper, error) {
	os.Remove("./foo.db")
	db, err := sql.Open("sqlite3", "./foo.db")
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE leaderboards (
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		leaderboard VARCHAR(64) UNIQUE,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE scores (
		leaderboard VARCHAR(64),
		user VARCHAR(64),
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		score INT,
		PRIMARY KEY(leaderboard, user)
	)`)

	if err != nil {
		log.Fatal(err)
	}
	return StorageWrapper{
		db,
	}, err

}

func (sw *StorageWrapper) Close() error {
	return sw.db.Close()
}

func (sw *StorageWrapper) NewLeaderBoard() (uuid.UUID, error) {
	leaderboard := uuid.New()
	_, err := sw.db.Exec("INSERT INTO leaderboards (leaderboard) VALUES (?)", leaderboard)
	return leaderboard, err
}

func (sw *StorageWrapper) UpdateScore(leaderboard uuid.UUID, user string, score int) error {
	_, err := sw.db.Exec(`
		INSERT INTO scores (leaderboard, user, score)
		VALUES (?1, ?2, ?3)
		ON CONFLICT (leaderboard, user) DO
		UPDATE SET score=excluded.score;
		`, leaderboard, user, score)
	return err
}

func (sw *StorageWrapper) GetLeaderboard(leaderboard uuid.UUID) ([]Entry, error) {
	rows, err := sw.db.Query(`SELECT user, score FROM scores WHERE leaderboard=?`, leaderboard)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Entry{}

	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.User, &e.Score); err != nil {
			return entries, err
		}
		entries = append(entries, e)

	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, err

}
