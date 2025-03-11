package main

import (
	"context"
	"errors"
	"fmt"
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

	id, db_err := app.st.newLeaderboard(ctx, input.UserID, input.Body)

	if db_err != nil {
		return nil, db_err
	}

	resp := &NewLeaderboardResponse{}
	resp.Body.Id = id
	return resp, db_err
}

func (app *App) postNewScore(ctx context.Context, input *struct {
	LeaderboardIDParam
	UserIDHeader
	NewSubmissionRequest
}) (*SubmissionResponse, error) {
	s_id, db_err := app.st.newSubmission(ctx, input.ID, input.UserID, input.Body.Score, input.Body.Link)
	if db_err != nil {
		return nil, db_err
	}

	app.cache.Remove(input.ID)

	return &SubmissionResponse{
		SubmissionResponseBody{
			s_id,
		},
	}, nil
}

func (app *App) getLeaderboard(ctx context.Context, input *struct {
	LastModified string `header:"If-Modified-Since"`
	LeaderboardIDParam
}) (*LeaderboardResponse, error) {
	last_updated, err := app.st.getLastUpdatedTime(ctx, input.ID)
	fmt.Println(last_updated)
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

	scores, db_err := app.st.getLeaderboard(ctx, input.ID)
	if db_err != nil {
		return nil, db_err
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
	submission_info, db_err := app.st.getSubmissionInfo(ctx, input.ID, input.SubmissionID)

	if db_err != nil {
		return nil, db_err
	}

	submission_info.LeaderboardID = input.ID
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
	new_score := input.Body.Score
	new_link := input.Body.Link

	_, db_err := app.st.updateSubmissionScore(ctx, input.ID, input.SubmissionID, new_score, new_link)

	if db_err != nil {
		return nil, db_err
	}

	app.cache.Remove(input.ID)
	resp := &SubmissionResponse{
		SubmissionResponseBody{
			input.SubmissionID,
		},
	}
	return resp, nil
}

func (app *App) GetSubmissionHistory(ctx context.Context, input *struct {
	LeaderboardIDParam
	SubmissionIDParam
}) (*HistoryResponse, error) {
	history, db_err := app.st.getSubmissionHistory(ctx, input.SubmissionID)

	if db_err != nil {
		return nil, db_err
	}

	resp := &HistoryResponse{
		Body: HistoryResponseBody{
			History: history,
		},
	}
	return resp, nil
}

func (app *App) AddSubmissionComment(ctx context.Context, input *struct {
	LeaderboardIDParam
	SubmissionIDParam
	UserIDHeader
	CommentSubmissionBody
}) (*SubmissionResponse, error) {
	db_err := app.st.addSubmissionComment(ctx, input.ID, input.SubmissionID, input.UserID, input.Body.Comment)

	if db_err != nil {
		return nil, db_err
	}

	resp := &SubmissionResponse{
		SubmissionResponseBody{
			ID: input.SubmissionID,
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
	count, db_err := app.st.verifyScore(ctx, input.ID, input.SubmissionID, input.UserID, input.Body.IsValid, input.Body.Comment)

	if count == 0 || db_err == pgx.ErrNoRows {
		return nil, huma.Error401Unauthorized("Not authorized to verify scores for this leaderboard.")
	}
	if db_err != nil {
		return nil, db_err
	}

	app.cache.Remove(input.ID)
	resp := &SubmissionResponse{
		SubmissionResponseBody{
			ID: input.SubmissionID,
		},
	}
	return resp, nil
}

func (app *App) getLeaderboardVerifiers(ctx context.Context, input *struct {
	LeaderboardIDParam
}) (*LeaderboardVerifiersResponse, error) {

	owners, db_err := app.st.getVerifiers(ctx, input.ID)

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

	info, db_err := app.st.getLeaderboardInfo(ctx, input.ID)

	if db_err != nil {
		return nil, db_err
	}

	resp := &LeaderboardInfoResponse{}
	resp.Body = info
	resp.Body.ID = input.ID
	return resp, nil
}

func (app *App) getAccountLeaderboards(ctx context.Context, input *struct {
	UserIDParam
}) (*AccountLeaderboardsResponse, error) {
	leaderboards, db_err := app.st.getAccountLeaderboards(ctx, input.UserID)
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

func (app *App) getAccountSubmissions(ctx context.Context, input *struct {
	UserIDParam
}) (*AccountSubmissionsResponse, error) {
	submissions, db_err := app.st.getAccountSubmissions(ctx, input.UserID)
	if db_err != nil {
		return nil, db_err
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
