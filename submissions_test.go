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

func getSubmissionDetailed(api humatest.TestAPI, leaderboard string, submission string) (DetailedSubmission, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s/submission/%s", leaderboard, submission))
	var lResp DetailedSubmission
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func TestGetSubmissionInfo(t *testing.T) {
	api := setupTestApi(t)
	id := createBasicLeaderboard(api, t, "testid")

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, postResp2.Code)

	var submitResponse SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &submitResponse)

	if submitInfo, getResp := getSubmissionDetailed(api, id, submitResponse.ID); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 10, submitInfo.Score)
		assert.Equal(t, "www.youtube.com", submitInfo.Link)
		assert.Equal(t, "testid2", submitInfo.Submitter.ID)
		assert.Equal(t, id, submitInfo.LeaderboardID)
		assert.Equal(t, "My First Leaderboard", submitInfo.LeaderboardDisplayName)
		assert.False(t, submitInfo.Verified)
	}
}

func TestUpdateSubmissionBecomesUnverified(t *testing.T) {
	api := setupTestApi(t)

	id := createBasicLeaderboard(api, t, "testid")

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}

	var submissionBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &submissionBody)
	updateResp := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/verify", id, submissionBody.ID),
		"UserID: testid",
		map[string]any{
			"is_valid": true,
		})
	assert.Equal(t, 200, updateResp.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.True(t, lResp.Scores[0].Verified)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	updateRespScore := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/score", id, newScoreBody.ID),
		"UserID: testid",
		map[string]any{
			"score": 100,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, updateRespScore.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 100, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}
}

func TestVerifyScoreNotOwner(t *testing.T) {
	api := setupTestApi(t)

	id := createBasicLeaderboard(api, t, "testid")
	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	updateResp := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/verify", id, newScoreBody.ID),
		"UserID: testid2",
		map[string]any{
			"is_valid": true,
		})
	assert.Equal(t, 401, updateResp.Code)

}

func TestVerifyScore(t *testing.T) {
	api := setupTestApi(t)

	id := createBasicLeaderboard(api, t, "testid")

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	updateResp := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/verify", id, newScoreBody.ID),
		"UserID: testid",
		map[string]any{
			"is_valid": true,
		})
	assert.Equal(t, 200, updateResp.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.True(t, lResp.Scores[0].Verified)
	}

}

func TestUpdateScore(t *testing.T) {
	api := setupTestApi(t)

	id := createBasicLeaderboard(api, t, "testid")
	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Zero(t, len(lResp.Scores))
	}

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
			"link":  "www.youtube.com",
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	postResp3 := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/score", id, newScoreBody.ID),
		"UserID: testid2",
		map[string]any{
			"score": 11,
			"link":  "www.youtube.com/1",
		})
	assert.Equal(t, 200, postResp3.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
	}

	if submitInfo, getResp := getSubmissionDetailed(api, id, newScoreBody.ID); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 11, submitInfo.Score)
		assert.Equal(t, "www.youtube.com/1", submitInfo.Link)
		assert.Equal(t, "testid2", submitInfo.Submitter.ID)
		assert.Equal(t, id, submitInfo.LeaderboardID)
		assert.Equal(t, "My First Leaderboard", submitInfo.LeaderboardDisplayName)
		assert.False(t, submitInfo.Verified)
	}
}
