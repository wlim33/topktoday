package main

import (
	"context"
	"database/sql"

	"github.com/danielgtaylor/huma/v2"
)

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
type UpdateScoreBody struct {
	Body struct {
		Score int `json:"score" required:"true"`
	}
}

type NewLeaderboardBody struct {
	Body struct {
		DisplayName string `json:"name" example:"My Fist Leaderboard" doc:"Leaderboard display name"`
	}
}
type MessageResponse struct {
	Message string `json:"message" example:"All systems go!" doc:"Human readable message."`
}
type ErrorResponse struct {
	Error error `json:"error"`
}

type NewLeaderboardResponseBody struct {
	Id   string `json:"id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character leaderboard ID used for querying."`
	Name string `json:"name" example:"My First Leaderboard" doc:"Leaderboard display name."`
}

type NewLeaderboardResponse struct {
	Body NewLeaderboardResponseBody
}

type AccountLeaderboardsResponseBody struct {
	Leaderboards []LeaderboardInfo `json:"leaderboards"`
}

type AccountLeaderboardsResponse struct {
	Body AccountLeaderboardsResponseBody
}

type LeaderboardResponseBody struct {
	Scores []Entry `json:"scores"`
}

type LeaderboardResponse struct {
	Body LeaderboardResponseBody
}

type HealthCheckResponse struct {
	Body MessageResponse
}
type LeaderboardNameResponseBody struct {
	Name string `json:"name" example:"My Fist Leaderboard" doc:"Leaderboard display name."`
}

type SubmissionResponseBody struct {
	ID string `json:"submission_id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character submission ID used for querying."`
}

type LeaderboardNameResponse struct {
	Body LeaderboardNameResponseBody
}
type SubmissionResponse struct {
	Body SubmissionResponseBody
}

type LeaderboardPostResponse struct {
	Status int
}

func (app *App) postNewLeaderboard(ctx context.Context, input *struct {
	NewLeaderboardBody
	UserIDHeader
}) (*NewLeaderboardResponse, error) {
	raw_id, name, db_err := app.st.NewLeaderBoard(input.Body.DisplayName, input.UserID)
	if db_err != nil {
		return nil, db_err
	}

	leaderboard_id :=
		app.parser.encodeLeaderboardID(raw_id)
	resp := &NewLeaderboardResponse{}
	resp.Body.Id = leaderboard_id
	resp.Body.Name = name
	return resp, db_err
}

func (app *App) postNewScore(ctx context.Context, input *struct {
	LeaderboardIDParam
	UserIDHeader
	UpdateScoreBody
}) (*SubmissionResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	new_score := input.Body.Score
	s_id, db_err := app.st.NewSubmissionScore(leaderboard_id, input.UserID, new_score)
	if db_err != nil {
		return nil, db_err
	}
	submission_id :=
		app.parser.encodeSubmissionID(s_id)
	return &SubmissionResponse{
		SubmissionResponseBody{
			submission_id,
		},
	}, nil
}

func (app *App) getLeaderboard(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)

	scores, db_err := app.st.GetLeaderboard(leaderboard_id)
	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardResponse{}
	resp.Body.Scores = scores
	return resp, nil
}

func (app *App) getAccountLeaderboards(ctx context.Context, input *struct {
	UserIDParam
}) (*AccountLeaderboardsResponse, error) {
	leaderboards, db_err := app.st.GetUserLeaderboards(input.UserID)
	if db_err != nil {
		return nil, db_err
	}
	resp := &AccountLeaderboardsResponse{
		Body: AccountLeaderboardsResponseBody{
			leaderboards,
		},
	}
	return resp, nil
}

func (app *App) updateSubmissionScore(ctx context.Context, input *struct {
	LeaderboardIDParam
	SubmissionIDParam
	UpdateScoreBody
}) (*SubmissionResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	submission_id := app.parser.decodeSubmissionID(input.SubmissionID)
	new_score := input.Body.Score

	_, db_err := app.st.UpdateSubmissionScore(leaderboard_id, submission_id, new_score)

	if db_err != nil {
		return nil, db_err
	}

	resp := &SubmissionResponse{
		SubmissionResponseBody{
			input.SubmissionID,
		},
	}
	return resp, nil
}

func (app *App) VerifyScore(ctx context.Context, input *struct {
	LeaderboardIDParam
	SubmissionIDParam
	UserIDHeader
	VerifyScoreBody
}) (*SubmissionResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	submission_id := app.parser.decodeSubmissionID(input.SubmissionID)
	owner := input.UserID
	id, db_err := app.st.VerifyScore(leaderboard_id, submission_id, owner, input.Body.IsValid)

	if db_err != nil {
		if db_err == sql.ErrNoRows {
			return nil, huma.Error401Unauthorized("Not authorized to verify scores for this leaderboard.")
		}
		return nil, db_err
	}

	resp := &SubmissionResponse{
		SubmissionResponseBody{
			app.parser.encodeSubmissionID(id),
		},
	}
	return resp, nil
}

func (app *App) getLeaderboardName(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardNameResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)

	display_name, db_err := app.st.GetLeaderboardName(leaderboard_id)

	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardNameResponse{}
	resp.Body.Name = display_name
	return resp, nil
}

func (app *App) healthCheck(ctx context.Context, input *struct {
}) (*HealthCheckResponse, error) {

	resp := &HealthCheckResponse{}
	resp.Body.Message = VERSION
	return resp, nil
}
