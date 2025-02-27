package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/sqids/sqids-go"
	"github.com/stretchr/testify/assert"
)

func setupTestApi(t humatest.TB) humatest.TestAPI {
	app := App{
		st: setupDB(os.Getenv("DB_URL")),
	}
	app.parser = NewParser()

	app.st.CreateTestUser("testid", "admin", "admin@admin.admin")
	app.st.CreateTestUser("testid2", "player2", "a@admin.admin")
	app.st.CreateTestUser("testid3", "player3", "s@admin.admin")
	_, api := humatest.New(t)
	app.addRoutes(api)
	return api
}

func getLeaderboard(api humatest.TestAPI, leaderboard_id string) (LeaderboardResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getAccountLeaderboards(api humatest.TestAPI, user_id string) (AccountLeaderboardsResponseBody, *httptest.ResponseRecorder) {
	getResp := api.Get(fmt.Sprintf("/account/%s/leaderboards", user_id))
	var lResp AccountLeaderboardsResponseBody
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

func TestUpdateSubmissionBecomesUnverified(t *testing.T) {
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

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", newResp.Id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, newResp.Id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}

	var submissionBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &submissionBody)
	updateResp := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/verify", newResp.Id, submissionBody.ID),
		"UserID: testid",
		map[string]any{
			"is_valid": true,
		})
	assert.Equal(t, 200, updateResp.Code)

	if lResp, getResp := getLeaderboard(api, newResp.Id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.True(t, lResp.Scores[0].Verified)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	updateRespScore := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/score", newResp.Id, newScoreBody.ID),
		"UserID: testid",
		map[string]any{
			"score": 100,
		})

	assert.Equal(t, 200, updateRespScore.Code)

	if lResp, getResp := getLeaderboard(api, newResp.Id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 100, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}
}

func TestVerifyScoreNotOwner(t *testing.T) {
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

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", newResp.Id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, newResp.Id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	updateResp := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/verify", newResp.Id, newScoreBody.ID),
		"UserID: testid2",
		map[string]any{
			"is_valid": true,
		})
	assert.Equal(t, 401, updateResp.Code)

}

func TestVerifyScore(t *testing.T) {
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

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", newResp.Id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
		})

	assert.Equal(t, 200, postResp2.Code)

	if lResp, getResp := getLeaderboard(api, newResp.Id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.False(t, lResp.Scores[0].Verified)
	}

	var newScoreBody SubmissionResponseBody
	json.Unmarshal(postResp2.Body.Bytes(), &newScoreBody)
	updateResp := api.Patch(
		fmt.Sprintf("/leaderboard/%s/submission/%s/verify", newResp.Id, newScoreBody.ID),
		"UserID: testid",
		map[string]any{
			"is_valid": true,
		})
	assert.Equal(t, 200, updateResp.Code)

	if lResp, getResp := getLeaderboard(api, newResp.Id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 10, lResp.Scores[0].Score)
		assert.True(t, lResp.Scores[0].Verified)
	}

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

func TestUpdateScore(t *testing.T) {
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

	postResp2 := api.Post(
		fmt.Sprintf("/leaderboard/%s/submission", id),
		"UserID: testid2",
		map[string]any{
			"score": 10,
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
		})
	assert.Equal(t, 200, postResp3.Code)

	if lResp, getResp := getLeaderboard(api, id); assert.Equal(t, 200, getResp.Code) {
		assert.Equal(t, 1, len(lResp.Scores))
		assert.Equal(t, 11, lResp.Scores[0].Score)
	}
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

func BenchmarkGetLeaderboard(b *testing.B) {
	api := setupTestApi(b)

	resp := api.Post("/leaderboard",
		"UserID: testid",
		map[string]any{
			"name": "test leaderboard",
		})

	var newResp NewLeaderboardResponseBody
	json.Unmarshal(resp.Body.Bytes(), &newResp)
	s, _ := sqids.New()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		id, _ := s.Encode([]uint64{uint64(n)})

		api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			"UserID: testid3",
			map[string]any{
				"score": n,
			})

		getLeaderboard(api, newResp.Id)

	}
}
