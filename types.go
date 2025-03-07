package main

import "time"

type NotAuthorizedResponse struct {
	Status int
	Body   MessageResponse
}

type LeaderboardIDParam struct {
	ID string `path:"leaderboard_id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character leaderboard ID used for querying." required:"true"`
}
type SubmissionIDParam struct {
	SubmissionID string `path:"submission_id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character submission ID used for querying." required:"true"`
}
type UserIDParam struct {
	UserID string `path:"user_id" required:"true"`
}

type UserIDHeader struct {
	UserID string `header:"UserID" required:"true"`
}

type VerifyScoreBody struct {
	Body struct {
		IsValid bool `json:"is_valid" required:"true"`
	}
}
type NewSubmissionRequest struct {
	Body struct {
		Link  string `json:"link" required:"true"`
		Score int    `json:"score" required:"true"`
	}
}

type LinkAnonymousBody struct {
	Body struct {
		AnonID string `json:"anon_id"`
	}
}

type NewLeaderboardBody struct {
	Body LeaderboardConfig
}
type MessageResponseBody struct {
	Message string `json:"message" example:"All systems go!" doc:"Human readable message."`
}
type ErrorResponse struct {
	Error error `json:"error"`
}

type NewLeaderboardResponseBody struct {
	Id string `json:"id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character leaderboard ID used for querying."`
}

type LeaderboardVerifiersResponse struct {
	Body LeaderboardVerifiersResponseBody
}
type LeaderboardVerifiersResponseBody struct {
	Verifiers []User `json:"verifiers"`
}
type NewLeaderboardResponse struct {
	Body NewLeaderboardResponseBody
}

type AccountLeaderboardsResponseBody struct {
	Leaderboards []LeaderboardInfo `json:"leaderboards"`
}

type AccountSubmissionsResponseBody struct {
	Submissions []DetailedSubmission `json:"submissions"`
}

type AccountSubmissionsResponse struct {
	Body AccountSubmissionsResponseBody
}

type AccountLeaderboardsResponse struct {
	Body AccountLeaderboardsResponseBody
}

type LeaderboardResponseBody struct {
	Scores []Ranking `json:"scores"`
}

type LeaderboardResponse struct {
	Status       int
	LastModified time.Time `header:"Last-Modified"`
	Body         *LeaderboardResponseBody
}

type MessageResponse struct {
	Body MessageResponseBody
}

type SubmissionResponseBody struct {
	ID string `json:"submission_id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character submission ID used for querying."`
}

type SubmissionInfoResponse struct {
	Body DetailedSubmission
}

type SubmissionInfoResponseBody struct {
	Score                  int    `json:"score" example:"12" doc:"Current score of submission."`
	LeaderboardID          string `json:"leaderboard_id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character leaderboard ID used for querying."`
	LeaderboardDisplayName string `json:"leaderboard_title" example:"My First Leaderboard" doc:"Leaderboard title for associated submission."`
	LatestLink             string `json:"link" example:"https://www.youtube.com/watch?v=rdx0TPjX1qE" doc:"Latest link for this submission."`
	Verified               bool   `json:"verified" example:"true" doc:"Current verification status."`
	SubmitterID            string `json:"submitter_id" example:"hxLZ9Efck" doc:"Submitter id."`
	SubmitterUsername      string `json:"username" example:"greensuigi" doc:"Submitter username."`
}

type LeaderboardInfoResponse struct {
	Body LeaderboardInfo
}
type SubmissionResponse struct {
	Body SubmissionResponseBody
}

type LeaderboardPostResponse struct {
	Status int
}
