package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

func (app *App) postNewLeaderboard(ctx context.Context, input *struct {
	NewLeaderboardBody
	UserIDHeader
}) (*NewLeaderboardResponse, error) {
	count, err := app.st.getActiveLeaderboardCount(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	if customer, ok := ctx.Value(CUSTOMER_CONTEXT_KEY).(*CustomerInfo); ok {
		api_err := app.sendUsageUpdate(customer.subscription_id, count)
		if api_err != nil {
			return nil, api_err
		}
		if count > 50 {
			return nil, errors.New("Reached new leaderboard limit -- upgrade your account or close an existing leaderboard.")
		}
	} else {
		if count > 5 {
			return nil, errors.New("Reached new leaderboard limit -- upgrade your account or close an existing leaderboard.")
		}
	}

	raw_id, db_err := app.st.newLeaderboard(ctx, input.UserID, input.Body)
	if db_err != nil {
		return nil, db_err
	}

	leaderboard_id := app.parser.encodeLeaderboardID(raw_id)
	resp := &NewLeaderboardResponse{}
	resp.Body.Id = leaderboard_id
	return resp, db_err
}

func (app *App) postNewScore(ctx context.Context, input *struct {
	LeaderboardIDParam
	UserIDHeader
	NewSubmissionRequest
}) (*SubmissionResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	s_id, db_err := app.st.newSubmission(ctx, leaderboard_id, input.UserID, input.Body.Score, input.Body.Link)
	if db_err != nil {
		return nil, db_err
	}

	app.cache.Remove(input.ID)

	submission_id :=
		app.parser.encodeSubmissionID(s_id)
	return &SubmissionResponse{
		SubmissionResponseBody{
			submission_id,
		},
	}, nil
}

func (app *App) getLeaderboard(ctx context.Context, input *struct {
	LastModified string `header:"If-Modified-Since"`
	LeaderboardIDParam
}) (*LeaderboardResponse, error) {

	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	last_updated, err := app.st.getLastUpdatedTime(ctx, leaderboard_id)
	if err != nil {
		return nil, err
	}
	if len(input.LastModified) != 0 {
		headerIfModifiedSince, header_parse_err := time.Parse(http.TimeFormat, input.LastModified)
		if header_parse_err != nil {
			return nil, header_parse_err
		}
		if headerIfModifiedSince.Compare(last_updated) > 0 {
			return &LeaderboardResponse{
				Status:       http.StatusNotModified,
				LastModified: last_updated,
			}, nil
		}
	}

	if cached_resp, ok := app.cache.Get(input.ID); ok {
		return cached_resp, nil
	}

	scores, db_err := app.st.getLeaderboard(ctx, leaderboard_id)
	if db_err != nil {
		return nil, db_err
	}

	for i := range len(scores) {
		scores[i].ID = app.parser.encodeSubmissionID(uint64(scores[i].rawID))
	}

	resp := &LeaderboardResponse{Status: 200}
	resp.Body = &LeaderboardResponseBody{
		Scores: scores,
	}
	app.cache.Add(input.ID, resp)
	return resp, nil
}

func (app *App) getSubmission(ctx context.Context, input *struct {
	LeaderboardIDParam
	SubmissionIDParam
}) (*SubmissionInfoResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	submission_id := app.parser.decodeSubmissionID(input.SubmissionID)

	submission_info, db_err := app.st.getSubmissionInfo(ctx, leaderboard_id, submission_id)

	if db_err != nil {
		return nil, db_err
	}

	submission_info.LeaderboardID = app.parser.encodeLeaderboardID(uint64(submission_info.rawLeaderboardID))
	resp := &SubmissionInfoResponse{
		submission_info,
	}
	return resp, nil
}

func (app *App) updateSubmission(ctx context.Context, input *struct {
	LeaderboardIDParam
	SubmissionIDParam
	NewSubmissionRequest
}) (*SubmissionResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)
	submission_id := app.parser.decodeSubmissionID(input.SubmissionID)
	new_score := input.Body.Score
	new_link := input.Body.Link

	_, db_err := app.st.updateSubmissionScore(ctx, leaderboard_id, submission_id, new_score, new_link)

	if db_err != nil {
		return nil, db_err
	}

	app.cache.Remove(input.ID)
	resp := &SubmissionResponse{
		SubmissionResponseBody{
			app.parser.encodeSubmissionID(submission_id),
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
	id, db_err := app.st.verifyScore(ctx, leaderboard_id, submission_id, owner, input.Body.IsValid)

	if db_err != nil {
		if db_err == pgx.ErrNoRows {
			return nil, huma.Error401Unauthorized("Not authorized to verify scores for this leaderboard.")
		}
		return nil, db_err
	}

	app.cache.Remove(input.ID)
	resp := &SubmissionResponse{
		SubmissionResponseBody{
			app.parser.encodeSubmissionID(id),
		},
	}
	return resp, nil
}

func (app *App) getLeaderboardVerifiers(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardVerifiersResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)

	owners, db_err := app.st.getVerifiers(ctx, leaderboard_id)

	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardVerifiersResponse{
		Body: LeaderboardVerifiersResponseBody{
			owners,
		},
	}
	return resp, nil
}

func (app *App) getLeaderboardInfo(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardInfoResponse, error) {
	leaderboard_id := app.parser.decodeLeaderboardID(input.ID)

	info, db_err := app.st.getLeaderboardInfo(ctx, leaderboard_id)

	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardInfoResponse{}
	resp.Body = info
	return resp, nil
}

func (app *App) getAccountLeaderboards(ctx context.Context, input *struct {
	UserIDParam
}) (*AccountLeaderboardsResponse, error) {
	leaderboards, db_err := app.st.getUserLeaderboards(ctx, input.UserID)
	if db_err != nil {
		return nil, db_err
	}

	for i := range len(leaderboards) {
		leaderboards[i].ID = app.parser.encodeLeaderboardID(uint64(leaderboards[i].rawID))

	}

	resp := &AccountLeaderboardsResponse{
		Body: AccountLeaderboardsResponseBody{
			leaderboards,
		},
	}
	return resp, nil
}

func (app *App) getAccountSubmissions(ctx context.Context, input *struct {
	UserIDParam
}) (*AccountSubmissionsResponse, error) {
	submissions, db_err := app.st.getUserSubmissions(ctx, input.UserID)
	if db_err != nil {
		return nil, db_err
	}

	for i := range len(submissions) {
		submissions[i].ID = app.parser.encodeSubmissionID(uint64(submissions[i].rawID))
		submissions[i].LeaderboardID = app.parser.encodeLeaderboardID(uint64(submissions[i].rawLeaderboardID))
	}

	resp := &AccountSubmissionsResponse{
		Body: AccountSubmissionsResponseBody{
			submissions,
		},
	}
	return resp, nil
}

func (app *App) linkAnonymousAccount(ctx context.Context, input *struct {
	UserIDHeader
	LinkAnonymousBody
}) (*MessageResponse, error) {
	db_err := app.st.linkAccounts(ctx, input.Body.AnonID, input.UserID)
	if db_err != nil {
		return nil, db_err
	}

	resp := &MessageResponse{
		Body: MessageResponseBody{
			Message: "Successfully linked anonymous account.",
		},
	}
	return resp, nil
}

func (app *App) healthCheck(ctx context.Context, input *struct {
}) (*MessageResponse, error) {

	resp := &MessageResponse{}
	resp.Body.Message = VERSION
	return resp, nil
}
