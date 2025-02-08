package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

type Entry struct {
	User          string    `json:"user"`
	Score         int       `json:"score"`
	TimeSubmitted time.Time `json:"submitted_at"`
}

type StorageWrapper struct {
	db *sql.DB
}

func setupDB() *sql.DB {
	connURL := os.Getenv("DB_URL")
	fmt.Println("connURL:", connURL)
	db, err := sql.Open("pgx", connURL)
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`DROP TABLE IF EXISTS leaderboards, scores`)

	_, err = db.Exec(`CREATE TABLE leaderboards (
		id SERIAL,
		leaderboard VARCHAR(64) UNIQUE,
		display_name VARCHAR(64),
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (id, leaderboard)
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE scores (
		leaderboard VARCHAR(64) REFERENCES leaderboards(leaderboard),
		username VARCHAR(64),
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		score INT,
		PRIMARY KEY(leaderboard, username)
	)`)

	if err != nil {
		log.Fatal(err)
	}
	return db
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
		display_name VARCHAR(64),
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

func (app *App) Close() error {
	return app.db.Close()
}

func (app *App) NewLeaderBoard(display_name string) (uuid.UUID, string, error) {
	leaderboard := uuid.New()
	_, err := app.db.Exec("INSERT INTO leaderboards (leaderboard, display_name) VALUES ($1, $2)", leaderboard, display_name)
	return leaderboard, display_name, err
}

func (app *App) UpdateScore(leaderboard uuid.UUID, user string, score int) error {
	_, err := app.db.Exec(`
		INSERT INTO scores (leaderboard, username, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (leaderboard, username) DO
		UPDATE SET score=excluded.score;
		`, leaderboard, user, score)
	return err
}

func (app *App) GetLeaderboard(leaderboard uuid.UUID) ([]Entry, error) {
	stmt, err := app.db.Prepare(`
		SELECT username, score, timestamp 
		FROM scores 
		WHERE leaderboard=$1 AND timestamp > (CURRENT_DATE - INTERVAL '1 days')
		ORDER BY 
			score DESC,
			timestamp DESC
		LIMIT 100;
		`)

	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(leaderboard)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Entry{}

	for rows.Next() {
		var e Entry
		if err := rows.Scan(&e.User, &e.Score, &e.TimeSubmitted); err != nil {
			return entries, err
		}
		entries = append(entries, e)
	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, err

}
