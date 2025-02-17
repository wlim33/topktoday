package main

import (
	"context"
	"net/http"
)

type LeaderboardIDParam struct {
	ID string `path:"id" example:"EfhxLZ9ck" minLength:"9" maxLength:"9" doc:"9 character leaderboard ID used for querying."`
}

type UpdateScoreBody struct {
	Body struct {
		User  string `json:"user"`
		Score int    `json:"score"`
	}
}

type NewLeaderboardBody struct {
	Body struct {
		DisplayName string `json:"name" example:"My Fist Leaderboard" doc:"Leaderboard display name"`
	}
}
type MessageResponse struct {
	Message string `json:"message"`
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

type LeaderboardResponseBody struct {
	Scores []Entry `json:"scores"`
}
type LeaderboardResponse struct {
	Body LeaderboardResponseBody
}

type LeaderboardNameResponseBody struct {
	Name string `json:"name" example:"My Fist Leaderboard" doc:"Leaderboard display name."`
}
type LeaderboardNameResponse struct {
	Body LeaderboardNameResponseBody
}

type LeaderboardPostResponse struct {
	Status int
}

func (app *App) postNewLeaderboard(ctx context.Context, input *struct {
	NewLeaderboardBody
}) (*NewLeaderboardResponse, error) {
	leaderboard_id, name, db_err := app.NewLeaderBoard(input.Body.DisplayName)
	if db_err != nil {
		return nil, db_err
	}

	display_id, _ := app.s.Encode([]uint64{leaderboard_id})
	resp := &NewLeaderboardResponse{}
	resp.Body.Id = display_id
	resp.Body.Name = name
	return resp, db_err
}

func (app *App) postNewScore(ctx context.Context, input *struct {
	LeaderboardIDParam
	UpdateScoreBody
}) (*LeaderboardPostResponse, error) {
	leaderboard_id := app.s.Decode(input.ID)[0]
	user := input.Body.User
	new_score := input.Body.Score
	if db_err := app.UpdateScore(leaderboard_id, user, new_score); db_err != nil {
		return nil, db_err
	}
	return &LeaderboardPostResponse{
		http.StatusOK,
	}, nil
}

func (app *App) getLeaderboard(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardResponse, error) {
	leaderboard_id := app.s.Decode(input.ID)[0]

	scores, db_err := app.GetLeaderboard(leaderboard_id)
	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardResponse{}
	resp.Body.Scores = scores
	return resp, nil
}
func (app *App) getLeaderboardName(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardNameResponse, error) {
	leaderboard_id := app.s.Decode(input.ID)[0]
	display_name, db_err := app.GetLeaderboardName(leaderboard_id)
	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardNameResponse{}
	resp.Body.Name = display_name
	return resp, nil
}
