//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
)

func getAccountLeaderboards(api humatest.TestAPI, user_id string) (AccountLeaderboardsResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/account/%s/leaderboards", user_id))
	var lResp AccountLeaderboardsResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getAccountSubmissions(api humatest.TestAPI, user_id string) (AccountSubmissionsResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/account/%s/submissions", user_id))
	var lResp AccountSubmissionsResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func TestGetUserLeaderboards(t *testing.T) {
	api := setupTestApi(t)

	resp := api.Post("/leaderboard",
		"UserID: testid",
		map[string]any{
			"name": "test name",
		})
	assert.Equal(t, 200, resp.Code)
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	assert.Equal(t, "test name", newResp.Name)

	if lResp, getResp := getAccountLeaderboards(api, "testid"); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Leaderboards))
		assert.Equal(t, "test name", lResp.Leaderboards[0].Title)
	}

	resp2 := api.Post("/leaderboard",
		"UserID: testid",
		map[string]any{
			"name": "test name 2",
		})
	assert.Equal(t, 200, resp2.Code)
	var newResp2 NewLeaderboardResponseBody
	json.Unmarshal(resp2.Body.Bytes(), &newResp2)
	assert.Equal(t, "test name 2", newResp2.Name)

	if lResp, getResp := getAccountLeaderboards(api, "testid"); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(lResp.Leaderboards))
	}
}

func TestGetUserSubmissions(t *testing.T) {
	api := setupTestApi(t)

	resp := api.Post("/leaderboard",
		"UserID: testid",
		map[string]any{
			"name": "test name",
		})
	assert.Equal(t, 200, resp.Code)
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	assert.Equal(t, "test name", newResp.Name)

	subResp := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", newResp.Id),
		"UserID: testid",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, subResp.Code)
	if submissionResp, getResp := getAccountSubmissions(api, "testid"); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(submissionResp.Submissions))
		assert.Equal(t, 10, submissionResp.Submissions[0].Score)
	}

	submissionResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", newResp.Id),
		"UserID: testid",
		map[string]any{
			"score": 11,
			"link":  "www.youtube.com/1",
		})

	assert.Equal(t, 200, submissionResp2.Code)

	if submissionResp, getResp := getAccountSubmissions(api, "testid"); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(submissionResp.Submissions))
		assert.Equal(t, 10, submissionResp.Submissions[1].Score)
		assert.Equal(t, 11, submissionResp.Submissions[0].Score)
	}

}

func TestLinkAnonymousAccount(t *testing.T) {
	api := setupTestApi(t)

	resp := api.Post("/leaderboard",
		"UserID: anon",
		map[string]any{
			"name": "test name",
		})
	assert.Equal(t, 200, resp.Code)
	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	assert.Equal(t, "test name", newResp.Name)

	id := newResp.Id

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Zero(t, len(lResp.Scores))
	}

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: anon",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, postResp2.Code)

	linkResponse := api.Post(
		"/account/link_anonymous",
		"UserID: testid",
		map[string]any{
			"anon_id": "anon",
		})

	assert.Equal(t, 200, linkResponse.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, "testid", lResp.Scores[0].User.ID)
		assert.Equal(t, "testid", lResp.Scores[0].User.ID)
	}
}
