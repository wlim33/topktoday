package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Storage struct {
	db *pgxpool.Pool
}

type Ranking struct {
	rawID int

	User          `json:"user"`
	ID            string    `json:"id"`
	Score         int       `json:"score"`
	TimeSubmitted time.Time `json:"submitted_at"`
	Verified      bool      `json:"verified"`
}

type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username" example:"greensuigi" doc:"Submitter username."`
	TimeAdded time.Time `json:"added_at"`
}

type LeaderboardInfo struct {
	rawID int

	ID          string    `json:"id"`
	Title       string    `json:"title" example:"My First Leaderboard" doc:"Leaderboard title for associated submission."`
	Verifiers   []User    `json:"verifiers,omitempty"`
	TimeCreated time.Time `json:"created_at"`
}

type DetailedSubmission struct {
	rawID            int
	rawLeaderboardID int

	Link                   string    `json:"link,omitempty" example:"https://www.youtube.com/watch?v=rdx0TPjX1qE" doc:"Latest link for this submission."`
	ID                     string    `json:"id,omitempty"`
	Score                  int       `json:"score" example:"12" doc:"Current score of submission."`
	LeaderboardID          string    `json:"leaderboard_id" example:"EfhxLZ9ck" doc:"9 character leaderboard ID used for querying."`
	LeaderboardDisplayName string    `json:"leaderboard_title" example:"My First Leaderboard" doc:"Leaderboard title for associated submission."`
	Submitter              *User     `json:"submitted_by,omitempty"`
	TimeCreated            time.Time `json:"last_submitted"`
	Verified               bool      `json:"verified" example:"true" doc:"Current verification status."`
}

func setupDB(ctx context.Context, connURL string) Storage {
	db, err := pgxpool.New(ctx, connURL)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Connected to db successfully.")

	}

	db.Exec(ctx, `DROP TABLE IF EXISTS leaderboards, submissions, owners, verifiers`)

	_, err = db.Exec(ctx, `CREATE TABLE leaderboards (
		id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
		created_by TEXT REFERENCES "user"(id),
		display_name TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(id, created_by)
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(ctx, `CREATE TABLE submissions (
		id INT GENERATED ALWAYS AS IDENTITY UNIQUE,
		leaderboard INT REFERENCES leaderboards(id),
		userid TEXT REFERENCES "user"(id),
		link TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		score NUMERIC NOT NULL,
		verified BOOLEAN NOT NULL DEFAULT FALSE,
		PRIMARY KEY(id, leaderboard, userid)
	)`)

	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(ctx, `CREATE TABLE verifiers (
		leaderboard INT REFERENCES leaderboards(id),
		userid TEXT REFERENCES "user"(id),
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY(leaderboard, userid)
	)`)

	if err != nil {
		log.Fatal(err)
	}
	return Storage{db}
}

func (st Storage) Close() {
	st.db.Close()
}

func (st Storage) newLeaderboard(ctx context.Context, display_name string, user_id string) (uint64, string, error) {
	row := st.db.QueryRow(ctx, `
		WITH ins_leaderboard AS (
			INSERT INTO leaderboards(display_name, created_by) 
			VALUES ($1, $2)
			RETURNING id
		)
		INSERT INTO verifiers(leaderboard, userid)
		SELECT id, $2
		FROM ins_leaderboard
		RETURNING verifiers.leaderboard
		`, display_name, user_id)
	var leaderboard_id int
	err := row.Scan(&leaderboard_id)
	if err != nil {
		log.Println(err)
	}
	return uint64(leaderboard_id), display_name, err
}

func (st Storage) verifyScore(ctx context.Context, leaderboard uint64, submission uint64, owner string, is_valid bool) (uint64, error) {
	var submission_id uint64
	err := st.db.QueryRow(ctx, `
		UPDATE submissions
		SET verified=$4
		WHERE leaderboard=$1 AND id=$2 AND EXISTS(SELECT 1 FROM verifiers WHERE leaderboard=$1 AND userid=$3)
		RETURNING id
		`, leaderboard, submission, owner, is_valid).Scan(&submission_id)
	if err != nil {
		return 0, err
	}
	return submission_id, nil
}

func (st Storage) getSubmissionInfo(ctx context.Context, leaderboard uint64, submission uint64) (DetailedSubmission, error) {
	var submissionInfo DetailedSubmission
	var submitter User
	err := st.db.QueryRow(ctx, `
		SELECT submissions.created_at, submissions.score, submissions.link, submissions.leaderboard, leaderboards.display_name, "user".name, "user".id, submissions.verified
		FROM submissions
		LEFT JOIN leaderboards
		ON leaderboards.id=submissions.leaderboard
		LEFT JOIN "user"
		ON "user".id=submissions.userid
		WHERE submissions.leaderboard=$1 AND submissions.id=$2
		`, leaderboard, submission).Scan(&submissionInfo.TimeCreated,
		&submissionInfo.Score,
		&submissionInfo.Link,
		&submissionInfo.rawLeaderboardID,
		&submissionInfo.LeaderboardDisplayName,
		&submitter.Username,
		&submitter.ID,
		&submissionInfo.Verified)
	if err != nil {
		return submissionInfo, err
	}
	submissionInfo.Submitter = &submitter
	return submissionInfo, nil
}

func (st Storage) newSubmission(ctx context.Context, leaderboard uint64, user string, score int, link string) (uint64, error) {
	var submission_id uint64
	err := st.db.QueryRow(ctx, `
		INSERT INTO submissions (leaderboard, userid, score, link)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`, leaderboard, user, score, link).Scan(&submission_id)
	if err != nil {
		return 0, err
	}
	return submission_id, nil
}

func (st Storage) updateSubmissionScore(ctx context.Context, leaderboard uint64, submission uint64, score int, link string) (uint64, error) {
	var submission_id uint64

	err := st.db.QueryRow(ctx, `
		UPDATE submissions
		SET
			score=$3,
			link=$4,
			verified=FALSE
		WHERE leaderboard=$1 AND id=$2
		RETURNING id;
		`, leaderboard, submission, score, link).Scan(&submission_id)

	if err != nil {
		return 0, err
	}
	return submission_id, nil
}

func (st Storage) getLeaderboardName(ctx context.Context, leaderboard uint64) (string, error) {
	row := st.db.QueryRow(ctx, `
		SELECT display_name
		FROM leaderboards 
		WHERE id=$1 AND created_at > (CURRENT_DATE - INTERVAL '1 days')
		`, leaderboard)

	var display_name string
	err := row.Scan(&display_name)
	if err != nil {
		return "", err
	}
	return display_name, nil
}

func (st Storage) getVerifiers(ctx context.Context, leaderboard_id uint64) ([]User, error) {
	rows, err := st.db.Query(ctx, `
		SELECT "user".id, "user".name, verifiers.added_at
		FROM verifiers
		LEFT JOIN "user"
		ON "user".id=verifiers.userid
		WHERE leaderboard=$1
		ORDER BY 
			added_at DESC
		LIMIT 25
		`, leaderboard_id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	owners := []User{}

	for rows.Next() {
		var owner User
		if err := rows.Scan(&owner.ID, &owner.Username, &owner.TimeAdded); err != nil {
			return owners, err
		}
		owners = append(owners, owner)
	}
	if err = rows.Err(); err != nil {
		return owners, err
	}
	return owners, err
}

func (st Storage) getUserLeaderboards(ctx context.Context, user_id string) ([]LeaderboardInfo, error) {
	rows, err := st.db.Query(ctx, `
		SELECT id, display_name, created_at
		FROM leaderboards
		WHERE created_by=$1
		ORDER BY 
			created_at DESC
		LIMIT 25
		`, user_id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	leaderboards := []LeaderboardInfo{}

	for rows.Next() {
		var li LeaderboardInfo
		if err := rows.Scan(&li.rawID, &li.Title, &li.TimeCreated); err != nil {
			return leaderboards, err
		}
		leaderboards = append(leaderboards, li)
	}
	if err = rows.Err(); err != nil {
		return leaderboards, err
	}
	return leaderboards, err
}

func (st Storage) getUserSubmissions(ctx context.Context, user_id string) ([]DetailedSubmission, error) {
	rows, err := st.db.Query(ctx, `
		SELECT submissions.id, leaderboards.display_name, submissions.created_at, submissions.score, leaderboards.id
		FROM submissions
		LEFT JOIN leaderboards
		ON submissions.leaderboard=leaderboards.id
		WHERE submissions.userid=$1
		ORDER BY 
			created_at DESC
		LIMIT 25
		`, user_id)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	submissions := []DetailedSubmission{}

	for rows.Next() {
		var s DetailedSubmission
		if err := rows.Scan(&s.rawID, &s.LeaderboardDisplayName, &s.TimeCreated, &s.Score, &s.rawLeaderboardID); err != nil {
			return submissions, err
		}
		submissions = append(submissions, s)
	}
	if err = rows.Err(); err != nil {
		return submissions, err
	}
	return submissions, err
}

func (st Storage) getLeaderboard(ctx context.Context, leaderboard uint64) ([]Ranking, error) {
	rows, err := st.db.Query(ctx, `
		SELECT submissions.userid, submissions.score, submissions.created_at, submissions.verified, submissions.id, "user".name
		FROM submissions 
		LEFT JOIN "user"
		ON "user".id = submissions.userid
		WHERE submissions.leaderboard=$1 AND submissions.created_at > (CURRENT_DATE - INTERVAL '1 days')
		ORDER BY 
			submissions.score DESC,
			submissions.created_at DESC
		LIMIT 25
		`, leaderboard)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Ranking{}

	for rows.Next() {
		var e Ranking
		var user User
		if err := rows.Scan(&user.ID, &e.Score, &e.TimeSubmitted, &e.Verified, &e.rawID, &user.Username); err != nil {
			return entries, err
		}
		e.User = user
		entries = append(entries, e)
	}
	if err = rows.Err(); err != nil {
		return entries, err
	}
	return entries, err
}

func (st Storage) linkAccounts(ctx context.Context, anon_id string, user_id string) error {

	tx, err := st.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)
	_, tx_err := tx.Exec(ctx, `
		UPDATE verifiers
		SET userid=$2
		WHERE userid=$1 AND EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND "isAnonymous"=true)
		`, anon_id, user_id)
	if tx_err != nil {
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `
		UPDATE leaderboards
		SET created_by=$2
		WHERE created_by=$1 AND EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND "isAnonymous"=true)
		`, anon_id, user_id)
	if tx_err != nil {
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `
		UPDATE submissions
		SET userid=$2
		WHERE userid=$1 AND EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND "isAnonymous"=true)
		`, anon_id, user_id)
	if tx_err != nil {
		return tx_err
	}

	commit_err := tx.Commit(ctx)
	return commit_err
}

func (st Storage) createTestUser(ctx context.Context, id string, name string, email string, is_anonymous bool) error {
	_, err := st.db.Exec(ctx, `
		INSERT INTO "user"(id, name, email, "emailVerified", "createdAt", "updatedAt", "isAnonymous")
		VALUES ($1, $2, $3, FALSE, NOW(), NOW(), $4)
		ON CONFLICT DO NOTHING;
		`, id, name, email, is_anonymous)
	return err
}
