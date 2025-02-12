package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Entry struct {
	User          string    `json:"user"`
	Score         int       `json:"score"`
	TimeSubmitted time.Time `json:"submitted_at"`
}

type StorageWrapper struct {
	db *sql.DB
}

func setupDB(connURL string) *sql.DB {
	db, err := sql.Open("pgx", connURL)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Connected to db successfully.")

	}

	db.Exec(`DROP TABLE IF EXISTS leaderboards, scores`)

	_, err = db.Exec(`CREATE TABLE leaderboards (
		id SERIAL PRIMARY KEY,
		display_name VARCHAR(64),
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE scores (
		leaderboard INT REFERENCES leaderboards(id),
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

func (app *App) Close() error {
	return app.db.Close()
}

func (app *App) NewLeaderBoard(display_name string) (uint64, string, error) {
	row := app.db.QueryRow(`
		INSERT INTO 
			leaderboards (display_name) 
		VALUES 
			($1)
		RETURNING id;
		`, display_name)
	var leaderboard_id int
	err := row.Scan(&leaderboard_id)
	if err != nil {
		log.Println(err)
	}
	return uint64(leaderboard_id), display_name, err
}

func (app *App) UpdateScore(leaderboard uint64, user string, score int) error {
	_, err := app.db.Exec(`
		INSERT INTO scores (leaderboard, username, score)
		VALUES ($1, $2, $3)
		ON CONFLICT (leaderboard, username) DO
		UPDATE SET score=excluded.score;
		`, leaderboard, user, score)
	return err
}

func (app *App) GetLeaderboardName(leaderboard uint64) (string, error) {
	row := app.db.QueryRow(`
		SELECT display_name
		FROM leaderboards 
		WHERE id=$1 AND timestamp > (CURRENT_DATE - INTERVAL '1 days')
		`)

	var display_name string
	err := row.Scan(&display_name)
	if err != nil {
		return "", err
	}
	return display_name, err

}
func (app *App) GetLeaderboard(leaderboard uint64) ([]Entry, error) {
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
