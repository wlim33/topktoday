package main

import (
	"context"
	"log"
	"time"

	_ "embed"

	"github.com/gofrs/uuid/v5"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
	pgconn "github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type DBConn interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, optionsAndArgs ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, optionsAndArgs ...any) pgx.Row
}
type DB struct {
	conn DBConn
}

type LeaderboardConfig struct {
	Title        string     `json:"title" example:"My First Leaderboard" doc:"Leaderboard title"`
	HighestFirst bool       `json:"highest_first" example:"true" doc:"If true, higher scores/times are ranked higher, e.g. highest score is first, second highest is second."`
	IsTime       bool       `json:"is_time" example:"false" doc:"If true, leaderboards scores are time values, e.g. 00:32"`
	NeedsVerify  bool       `json:"verify" example:"true" doc:"If true, submissions need to be verified before they show up on the leaderboard."`
	Stop         *time.Time `json:"stop,omitempty"  format:"date-time" example:"2024-09-05T14\:35" doc:"Datetime when the leaderboard closes. Times before the start value or empty mean the leaderboard accept submissions until the leaderboard is archived."`
	Start        time.Time  `json:"start" format:"date-time" example:"2024-09-05T14\:35" doc:"Datetime when the leaderboard opens. Default is at time of leaderboard creation."`
}

type HistoryEntry struct {
	Comment       string    `json:"comment"`
	TimeSubmitted time.Time `json:"submitted_at"`
	Author        User      `json:"author"`
	Action        string    `json:"action"`
}

type Ranking struct {
	User          `json:"user"`
	ID            uuid.UUID `json:"id"`
	Score         int       `json:"score"`
	TimeSubmitted time.Time `json:"submitted_at"`
	Verified      *bool     `json:"verified,omitempty"`
}

type User struct {
	ID        string     `json:"id"`
	Username  string     `json:"username" example:"greensuigi" doc:"Submitter username."`
	TimeAdded *time.Time `json:"added_at,omitempty"`
}

type LeaderboardInfo struct {
	ID          uuid.UUID `json:"id"`
	Verifiers   []User    `json:"verifiers,omitempty"`
	TimeCreated time.Time `json:"time_created"`
	LeaderboardConfig
}

type DetailedSubmission struct {
	Link                   string    `json:"link,omitempty" format:"uri" example:"https://www.youtube.com/watch?v=rdx0TPjX1qE" doc:"Latest link for this submission."`
	ID                     uuid.UUID `json:"id,omitempty"`
	Score                  int       `json:"score" example:"12" doc:"Current score of submission."`
	LeaderboardID          uuid.UUID `json:"leaderboard_id" example:"EfhxLZ9ck" doc:"9 character leaderboard ID used for querying."`
	LeaderboardDisplayName string    `json:"leaderboard_title" example:"My First Leaderboard" doc:"Leaderboard title for associated submission."`
	Submitter              *User     `json:"submitted_by,omitempty"`
	TimeCreated            time.Time `json:"last_submitted"`
	Verified               bool      `json:"verified" example:"true" doc:"Current verification status."`
}

//go:embed init.sql
var init_file string

func NewDBConn(ctx context.Context, connURL string) DB {
	dbconfig, err := pgxpool.ParseConfig(connURL)
	if err != nil {
		log.Fatal(err)
	}
	dbconfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {

		conn.Exec(ctx, `DROP TABLE IF EXISTS leaderboards, submissions, verifiers, submission_updates, customers;`)
		_, err = conn.Exec(ctx, init_file)
		if err != nil {
			log.Fatal(err)
		}
		pgxuuid.Register(conn.TypeMap())

		dt, err := conn.LoadType(ctx, "submission_action")
		if err != nil {
			log.Fatal(err)
		}

		conn.TypeMap().RegisterType(dt)

		return nil
	}
	db, err := pgxpool.NewWithConfig(ctx, dbconfig)

	if err != nil {
		log.Fatal(err)
	} else {
		log.Println("Connected to db successfully.")

	}

	return DB{db}
}

func (db DB) newLeaderboard(ctx context.Context, user_id string, config LeaderboardConfig) (uuid.UUID, error) {
	var leaderboard_id uuid.UUID
	err := db.conn.QueryRow(ctx, `
		WITH ins_leaderboard AS (
			INSERT INTO leaderboards(created_by, title, highest_first, is_time, start, stop, needs_verification) 
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		)
		INSERT INTO verifiers(leaderboard, userid)
		SELECT id, $1
		FROM ins_leaderboard
		RETURNING verifiers.leaderboard
		`, user_id, config.Title, config.HighestFirst, config.IsTime, config.Start, config.Stop, config.NeedsVerify).Scan(&leaderboard_id)

	return leaderboard_id, err
}

func (db DB) getSubmissionHistory(ctx context.Context, submission uuid.UUID) ([]HistoryEntry, error) {
	rows, err := db.conn.Query(ctx, `
		SELECT "user".id, "user".name, submission_updates.created_at, comment, action
		FROM submission_updates
		LEFT JOIN "user"
		ON "user".id=submission_updates.author
		WHERE submission_updates.submission=$1
		ORDER BY 
			submission_updates.created_at DESC
		LIMIT 25
		`, submission)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := []HistoryEntry{}

	for rows.Next() {
		var entry HistoryEntry
		var author User
		if err := rows.Scan(&author.ID, &author.Username, &entry.TimeSubmitted, &entry.Comment, &entry.Action); err != nil {
			return nil, err
		}
		entry.Author = author
		history = append(history, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return history, err

}
func (db DB) addSubmissionComment(ctx context.Context, leaderboard uuid.UUID, submission uuid.UUID, author string, comment string) error {

	_, err := db.conn.Exec(ctx, `
		INSERT INTO submission_updates(submission, author, comment, action)
		SELECT $2, $3, $4, 'comment'
		WHERE ((EXISTS(SELECT 1 FROM leaderboards WHERE leaderboards.id=$1 AND leaderboards.needs_verification IS TRUE)
				AND EXISTS(SELECT 1 FROM verifiers WHERE verifiers.leaderboard=$1 AND verifiers.userid=$3)) 
			OR EXISTS(SELECT 1 FROM leaderboards WHERE leaderboards.id=$1 AND leaderboards.needs_verification IS FALSE)
			);
		`, leaderboard, submission, author, comment)

	if err != nil {
		log.Println(err)
	}
	return err
}

func (db DB) verifyScore(ctx context.Context, leaderboard uuid.UUID, submission uuid.UUID, author string, is_valid bool, comment string) (int64, error) {
	result, err := db.conn.Exec(ctx, `
		WITH insert_history AS (
			INSERT INTO submission_updates(submission, author, comment, action)
			VALUES ($2, $3, $5, CAST(CASE WHEN $4 Then 'validate' ELSE 'invalidate' END AS submission_action))
		)
		UPDATE submissions
		SET verified=$4
		WHERE submissions.leaderboard=$1
			AND submissions.id=$2 
			AND (
				(EXISTS(SELECT 1 FROM leaderboards WHERE leaderboards.id=$1 AND leaderboards.needs_verification IS TRUE)
					AND EXISTS(SELECT 1 FROM verifiers WHERE verifiers.leaderboard=$1 AND verifiers.userid=$3)) 
				OR EXISTS(SELECT 1 FROM leaderboards WHERE leaderboards.id=$1 AND leaderboards.needs_verification IS FALSE)
			);
		`, leaderboard, submission, author, is_valid, comment)

	if err != nil {
		log.Println(err)
	}
	return result.RowsAffected(), nil
}

func (db DB) getSubmissionInfo(ctx context.Context, leaderboard uuid.UUID, submission uuid.UUID) (DetailedSubmission, error) {
	var submissionInfo DetailedSubmission
	var submitter User
	err := db.conn.QueryRow(ctx, `
		SELECT submissions.created_at, submissions.score, submissions.link, submissions.leaderboard, leaderboards.title, "user".name, "user".id, submissions.verified
		FROM submissions
		LEFT JOIN leaderboards
		ON leaderboards.id=submissions.leaderboard
		LEFT JOIN "user"
		ON "user".id=submissions.userid
		WHERE submissions.leaderboard=$1 AND submissions.id=$2
		`, leaderboard, submission).Scan(&submissionInfo.TimeCreated,
		&submissionInfo.Score,
		&submissionInfo.Link,
		&submissionInfo.LeaderboardID,
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

func (db DB) newSubmission(ctx context.Context, leaderboard uuid.UUID, user string, score int, link string) (uuid.UUID, error) {
	var submission_id uuid.UUID
	err := db.conn.QueryRow(ctx, `
		INSERT INTO submissions (leaderboard, userid, score, link)
		VALUES ($1, $2, $3, $4)
		RETURNING id
		`, leaderboard, user, score, link).Scan(&submission_id)
	if err != nil {
		log.Println(err)
	}
	return submission_id, nil
}

func (db DB) updateSubmissionScore(ctx context.Context, leaderboard uuid.UUID, submission uuid.UUID, score int, link string) (uuid.UUID, error) {
	var submission_id uuid.UUID

	err := db.conn.QueryRow(ctx, `
		UPDATE submissions
		SET
			score=$3,
			link=$4,
			verified=FALSE
		WHERE leaderboard=$1 AND id=$2
		RETURNING id;
		`, leaderboard, submission, score, link).Scan(&submission_id)

	if err != nil {
		return uuid.UUID{}, err
	}
	return submission_id, nil
}

func (db DB) getLeaderboardInfo(ctx context.Context, leaderboard uuid.UUID) (LeaderboardInfo, error) {
	var info LeaderboardInfo
	err := db.conn.QueryRow(ctx, `
		SELECT title, start, stop, is_time, needs_verification, highest_first, created_at
		FROM leaderboards 
		WHERE id=$1;
		`, leaderboard).Scan(&info.Title, &info.LeaderboardConfig.Start, &info.Stop, &info.IsTime, &info.NeedsVerify, &info.HighestFirst, &info.TimeCreated)

	if err != nil {
		return info, err
	}
	return info, nil
}

func (db DB) getVerifiers(ctx context.Context, leaderboard_id uuid.UUID) ([]User, error) {
	rows, err := db.conn.Query(ctx, `
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

func (db DB) getAccountLeaderboards(ctx context.Context, user_id string) ([]LeaderboardInfo, error) {
	rows, err := db.conn.Query(ctx, `
		SELECT id, title, created_at, start, stop
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
		if err := rows.Scan(&li.ID, &li.Title, &li.TimeCreated, &li.Start, &li.Stop); err != nil {
			return leaderboards, err
		}
		leaderboards = append(leaderboards, li)
	}
	if err = rows.Err(); err != nil {
		return leaderboards, err
	}
	return leaderboards, err
}

func (db DB) getAccountSubmissions(ctx context.Context, user_id string) ([]DetailedSubmission, error) {
	rows, err := db.conn.Query(ctx, `
		SELECT submissions.id, leaderboards.title, submissions.created_at, submissions.score, leaderboards.id
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
		if err := rows.Scan(&s.ID, &s.LeaderboardDisplayName, &s.TimeCreated, &s.Score, &s.LeaderboardID); err != nil {
			return submissions, err
		}
		submissions = append(submissions, s)
	}
	if err = rows.Err(); err != nil {
		return submissions, err
	}
	return submissions, err
}

func (db DB) getLeaderboard(ctx context.Context, leaderboard uuid.UUID) ([]Ranking, error) {
	rows, err := db.conn.Query(ctx, `
		WITH leaderboard_config(cutoff, highest_first, needs_verification) AS (
			SELECT
				CASE WHEN stop is NULL THEN NULL
				ELSE stop
				END, highest_first, needs_verification
			FROM leaderboards
			WHERE id=$1
		)
		SELECT submissions.userid, submissions.score, submissions.created_at, (CASE WHEN leaderboard_config.needs_verification THEN submissions.verified ELSE NULL END), submissions.id, "user".name
		FROM 
			(submissions LEFT JOIN "user"
				ON "user".id = submissions.userid), 
			leaderboard_config
		WHERE submissions.leaderboard=$1 AND (leaderboard_config.cutoff > submissions.created_at OR leaderboard_config.cutoff is NULL)
		ORDER BY 
			(CASE WHEN leaderboard_config.highest_first THEN submissions.score END) DESC,
			submissions.score ASC,
			submissions.created_at DESC
		LIMIT 100
		`, leaderboard)

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	entries := []Ranking{}

	for rows.Next() {
		var e Ranking
		var user User
		if err := rows.Scan(&user.ID, &e.Score, &e.TimeSubmitted, &e.Verified, &e.ID, &user.Username); err != nil {
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

func (db DB) linkAccounts(ctx context.Context, anon_id string, user_id string) error {

	tx, err := db.conn.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)
	_, tx_err := tx.Exec(ctx, `
		UPDATE verifiers
		SET userid=$2
		WHERE userid=$1 AND EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND "isAnonymous"=TRUE)
		`, anon_id, user_id)
	if tx_err != nil {
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `
		UPDATE leaderboards
		SET created_by=$2
		WHERE created_by=$1 AND EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND "isAnonymous"=TRUE)
		`, anon_id, user_id)
	if tx_err != nil {
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `
		UPDATE submissions
		SET userid=$2
		WHERE userid=$1 AND EXISTS(SELECT 1 FROM "user" WHERE id=$1 AND "isAnonymous"=TRUE)
		`, anon_id, user_id)
	if tx_err != nil {
		return tx_err
	}

	commit_err := tx.Commit(ctx)
	return commit_err
}

func (db DB) getActiveLeaderboardCount(ctx context.Context, user_id string) (int, error) {
	var rowCount int
	err := db.conn.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM leaderboards
		WHERE created_by=$1 AND (stop > NOW() OR stop is NULL)
		`, user_id).Scan(&rowCount)
	if err != nil {
		return -1, err
	}
	return rowCount, nil

}

func (db DB) createTestUser(ctx context.Context, id string, name string, email string, is_anonymous bool, customer_info CustomerInfo) error {

	tx, err := db.conn.Begin(ctx)
	if err != nil {
		log.Println(err)
		return err
	}

	defer tx.Rollback(ctx)
	_, tx_err := tx.Exec(ctx, `DELETE FROM customers WHERE customer_id=$1`, customer_info.id)
	if tx_err != nil {

		log.Println("Couldn't clear from customers table", tx_err)
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `DELETE FROM "session" WHERE "userId"=$1`, id)
	if tx_err != nil {
		log.Println("Couldn't clear from session table", tx_err)
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `DELETE FROM "user" WHERE name=$1 OR email=$2`, name, email)
	if tx_err != nil {
		log.Println("Couldn't clear from user table", tx_err)
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `
			INSERT INTO "user"(id, name, email, "emailVerified", "createdAt", "updatedAt", "isAnonymous")
			VALUES ($1, $2, $3, FALSE, NOW(), NOW(), $4)
		`, id, name, email, is_anonymous)
	if tx_err != nil {
		log.Println(tx_err)
		return tx_err
	}

	_, tx_err = tx.Exec(ctx, `
		INSERT INTO customers(userid, customer_id)
		VALUES ($1, $2) 
		`, id, customer_info.id)
	if tx_err != nil {
		log.Println(tx_err)
		return tx_err
	}
	commit_err := tx.Commit(ctx)
	return commit_err
}

func (db DB) getCustomer(ctx context.Context, id string) (CustomerInfo, error) {
	row := db.conn.QueryRow(ctx, `
		SELECT customer_id, status
		FROM customers
		WHERE userid=$1
		`, id)

	var customer CustomerInfo
	err := row.Scan(&customer.id, &customer.status)
	if err != nil {
		return customer, err
	}
	return customer, nil
}

func (db DB) updateSubscription(ctx context.Context, user_id string, attributes *CustomerAttributes) error {
	_, err := db.conn.Exec(ctx, `
		INSERT INTO customers(userid, customer_id, order_id, order_item_id, product_id, variant_id, user_name, user_email, status, status_formatted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (userid, customer_id) DO UPDATE 
		SET 
			order_id = excluded.order_id, 
			order_item_id = excluded.order_item_id, 
			product_id = excluded.product_id, 
			variant_id = excluded.variant_id, 
			user_name = excluded.user_name, 
			user_email = excluded.user_email, 
			status = excluded.status, 
			status_formatted = excluded.status_formatted
		`,
		user_id,
		attributes.CustomerID,
		attributes.OrderID,
		attributes.OrderItemID,
		attributes.ProductID,
		attributes.VariantID,
		attributes.CustomerName,
		attributes.CustomerEmail,
		attributes.Status,
		attributes.StatusFormatted)

	return err
}

func (db DB) getLastUpdatedTime(ctx context.Context, leaderboard_id uuid.UUID) (time.Time, error) {
	var lastUpdated time.Time
	err := db.conn.QueryRow(ctx, `
		SELECT last_updated
		FROM leaderboards
		WHERE id=$1
		`, leaderboard_id).Scan(&lastUpdated)

	return lastUpdated, err
}
