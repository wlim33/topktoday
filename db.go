package main

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Storage struct {
	db *sql.DB
}

type Entry struct {
	UserID        string    `json:"user"`
	DisplayName   string    `json:"display_name"`
	SubmissionID  string    `json:"id"`
	Score         int       `json:"score"`
	TimeSubmitted time.Time `json:"submitted_at"`
	Verified      bool      `json:"verified"`
}

type LeaderboardInfo struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"title"`
	TimeCreated time.Time `json:"created_at"`
}

func setupDB(connURL string) Storage {
	db, err := sql.Open("pgx", connURL)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Connected to db successfully.")

	}

	db.Exec(`DROP TABLE IF EXISTS leaderboards, submissions, apiKeys`)

	_, err = db.Exec(`CREATE TABLE leaderboards (
		id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
		owner TEXT REFERENCES "user"(id),
		display_name TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(id, owner)
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE submissions (
		id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
		leaderboard INT REFERENCES leaderboards(id),
		userid TEXT REFERENCES "user"(id),
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		score NUMERIC NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT FALSE,
		PRIMARY KEY(id, leaderboard, userid)
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE apiKeys (
		id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
		userid TEXT REFERENCES "user"(id),
		key TEXT NOT NULL,
		createdOn TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(id, userid)
	)`)

	if err != nil {
		log.Fatal(err)
	}
	return Storage{db}
}

func (st Storage) Close() error {
	return st.db.Close()
}

func (st Storage) NewLeaderBoard(display_name string, user_id string) (uint64, string, error) {
	row := st.db.QueryRow(`
		INSERT INTO 
			leaderboards (display_name, owner) 
		VALUES 
			($1, $2)
		RETURNING id;
		`, display_name, user_id)
	var leaderboard_id int
	err := row.Scan(&leaderboard_id)
	if err != nil {
		log.Println(err)
	}
	return uint64(leaderboard_id), display_name, err
}

func (st Storage) VerifyScore(leaderboard uint64, submission uint64, owner string, is_valid bool) (uint64, error) {
	var submission_id uint64
	err := st.db.QueryRow(`
		UPDATE submissions
		SET verified=$4
		WHERE leaderboard=$1 AND id=$2 AND EXISTS(SELECT 1 FROM leaderboards WHERE owner=$3)
		RETURNING id
		`, leaderboard, submission, owner, is_valid).Scan(&submission_id)
	if err != nil {
		return 0, err
	}
	return submission_id, nil

}

func (st Storage) NewSubmissionScore(leaderboard uint64, user string, score int) (uint64, error) {
	var submission_id uint64
	err := st.db.QueryRow(`
		INSERT INTO submissions (leaderboard, userid, score)
		VALUES ($1, $2, $3)
		RETURNING id
		`, leaderboard, user, score).Scan(&submission_id)
	if err != nil {
		return 0, err
	}
	return submission_id, nil
}

func (st Storage) UpdateSubmissionScore(leaderboard uint64, submission uint64, score int) (uint64, error) {
	var submission_id uint64

	err := st.db.QueryRow(`
		UPDATE submissions
		SET
			score=$3,
			verified=FALSE
		WHERE leaderboard=$1 AND id=$2
		RETURNING id;
		`, leaderboard, submission, score).Scan(&submission_id)

	if err != nil {
		return 0, err
	}
	return submission_id, nil
}

func (st Storage) GetLeaderboardName(leaderboard uint64) (string, error) {
	row := st.db.QueryRow(`
		SELECT display_name
		FROM leaderboards 
		WHERE id=$1 AND timestamp > (CURRENT_DATE - INTERVAL '1 days')
		`, leaderboard)

	var display_name string
	err := row.Scan(&display_name)
	if err != nil {
		return "", err
	}
	return display_name, nil
}

func (st Storage) GetUserLeaderboards(user_id string) ([]LeaderboardInfo, error) {
	rows, err := st.db.Query(`
		SELECT id, display_name, timestamp
		FROM leaderboards
		WHERE owner=$1
		ORDER BY 
			timestamp DESC
		`, user_id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	leaderboards := []LeaderboardInfo{}

	for rows.Next() {
		var li LeaderboardInfo
		if err := rows.Scan(&li.ID, &li.DisplayName, &li.TimeCreated); err != nil {
			return leaderboards, err
		}
		leaderboards = append(leaderboards, li)
	}
	if err = rows.Err(); err != nil {
		return leaderboards, err
	}
	return leaderboards, err
}

func (st Storage) GetLeaderboard(leaderboard uint64) ([]Entry, error) {
	stmt, err := st.db.Prepare(`
		SELECT submissions.userid, submissions.score, submissions.timestamp, submissions.verified, submissions.id, "user".name
		FROM submissions 
		LEFT JOIN "user"
		ON "user".id = submissions.userid
		WHERE submissions.leaderboard=$1 AND submissions.timestamp > (CURRENT_DATE - INTERVAL '1 days')
		ORDER BY 
			submissions.score DESC,
			submissions.timestamp DESC
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
		if err := rows.Scan(&e.UserID, &e.Score, &e.TimeSubmitted, &e.Verified, &e.SubmissionID, &e.DisplayName); err != nil {
			return entries, err
		}
		entries = append(entries, e)
	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, err
}

func (st Storage) CreateTestUser(id string, name string, email string) error {
	_, err := st.db.Exec(`
		INSERT INTO "user"(id, name, email, "emailVerified", "createdAt", "updatedAt")
		VALUES ($1, $2, $3, FALSE, NOW(), NOW())
		ON CONFLICT DO NOTHING;
		`, id, name, email)
	return err
}
