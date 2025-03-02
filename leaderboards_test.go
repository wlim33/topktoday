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

func getLeaderboard(api humatest.TestAPI, leaderboard_id string) (LeaderboardResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func TestDisplayName(t *testing.T) {

	api := setupTestApi(t)
	display_name := "Test Leaderboard Name"

	resp := api.Post("/leaderboard",
		"UserID: testid",
		map[string]any{
			"name": display_name,
		})

	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	assert.Equal(t, display_name, newResp.Name)

	id := newResp.Id
	assert.Equal(t, 200, resp.Code)

	getNameResp := api.Get(fmt.Sprintf("/leaderboard/%s/name", id))

	var lResp LeaderboardNameResponseBody
	json.Unmarshal(getNameResp.Body.Bytes(), &lResp)

	assert.Equal(t, display_name, lResp.Name)
}

func TestGetBadIDEmpty(t *testing.T) {
	api := setupTestApi(t)
	badID := ""
	_, getResp := getLeaderboard(api, badID)
	assert.Equal(t, 404, getResp.Code)
}

func TestGetBadIDTooShort(t *testing.T) {
	api := setupTestApi(t)
	badID := "123"
	_, getResp := getLeaderboard(api, badID)
	assert.Equal(t, 422, getResp.Code)
}

func TestGetBadIDTooLong(t *testing.T) {
	api := setupTestApi(t)
	badID := "iaersntaoirseoiaerstoieanrt"
	_, getResp := getLeaderboard(api, badID)
	assert.Equal(t, 422, getResp.Code)
}

func TestAddScores(t *testing.T) {
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

	id := newResp.Id

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Zero(t, len(lResp.Scores))
	}

	postResp := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 9,
		})

	assert.Equal(t, 200, postResp.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 9, lResp.Scores[0].Score)
	}

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.Equal(t, 9, lResp.Scores[1].Score)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	postResp3 := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/score", id, newScoreBody.ID),
		"UserID: testid2",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 11,
		})
	assert.Equal(t, 200, postResp3.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 2, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
		assert.Equal(t, 9, lResp.Scores[1].Score)
	}

	postResp4 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid3",
		map[string]any{
			"link":  "www.youtube.com",
			"score": 11,
		})

	assert.Equal(t, 200, postResp4.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 3, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
		assert.Equal(t, 11, lResp.Scores[1].Score)
		assert.Equal(t, 9, lResp.Scores[2].Score)
	}
}
