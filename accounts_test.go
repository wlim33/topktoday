//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
)

func getAccountLeaderboards(t *testing.T, api humatest.TestAPI, user_id string) (AccountLeaderboardsResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/account/%s/leaderboards", user_id))
	var lResp AccountLeaderboardsResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getAccountSubmissions(t *testing.T, api humatest.TestAPI, user_id string) (AccountSubmissionsResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/account/%s/submissions", user_id))
	var lResp AccountSubmissionsResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func TestGetUserLeaderboards(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {
		createBasicLeaderboard(t, api, users["player2"])

		if lResp, getResp := getAccountLeaderboards(t, api, users["player2"]); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 1, len(lResp.Leaderboards))
			assert.Equal(t, "My First Leaderboard", lResp.Leaderboards[0].Title)
		}

		createBasicLeaderboard(t, api, users["player2"])

		if lResp, getResp := getAccountLeaderboards(t, api, users["player2"]); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 2, len(lResp.Leaderboards))
		}
	})

}

func TestGetUserSubmissions(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {

		id := createBasicLeaderboard(t, api, users["Anonymous"])

		subResp := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"score": 10,
				"link":  "www.youtube.com",
			})

		assert.Equal(t, 200, subResp.Code)
		if submissionResp, getResp := getAccountSubmissions(t, api, users["player2"]); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 1, len(submissionResp.Submissions))
			assert.Equal(t, 10, submissionResp.Submissions[0].Score)
		}

		submissionResp2 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"score": 11,
				"link":  "www.youtube.com/1",
			})

		assert.Equal(t, 200, submissionResp2.Code)

		if submissionResp, getResp := getAccountSubmissions(t, api, users["player2"]); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 2, len(submissionResp.Submissions))
		}
	})

}

func TestLinkAnonymousAccount(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {

		id := createBasicLeaderboard(t, api, users["Anonymous"])

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Zero(t, len(lResp.Scores))
		}

		postResp2 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["Anonymous"]),
			map[string]any{
				"score": 10,
				"link":  "www.youtube.com",
			})

		assert.Equal(t, 200, postResp2.Code)

		linkResponse := api.Post(
			"/account/link_anonymous",
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"anon_id": users["Anonymous"],
			})

		assert.Equal(t, 200, linkResponse.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 1, len(lResp.Scores))
			assert.Equal(t, users["player2"], lResp.Scores[0].User.ID)
		}
	})
}

func TestActiveLeaderboardCount(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {

		for i := range 50 {
			resp := api.Post("/leaderboard",
				fmt.Sprintf("UserID: %s", users["player2"]),
				map[string]any{
					"title":                "My First Leaderboard",
					"highest_first":        true,
					"is_time":              true,
					"duration":             "00:01:00",
					"start":                "2020-03-05T18:54:00+00:00",
					"multiple_submissions": true,
				})
			assert.Equal(t, 200, resp.Code, "Failed to create %s leaderboards", i+1)

		}
	})
}
