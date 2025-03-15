//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
)

func getSubmissionHistory(t *testing.T, api humatest.TestAPI, leaderboard_id uuid.UUID, submission uuid.UUID) (HistoryResponseBody, *httptest.ResponseRecorder) {
	t.Helper()
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s/submission/%s/history", leaderboard_id, submission))
	var lResp HistoryResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getLeaderboardWithModifiedHeader(t *testing.T, api humatest.TestAPI, leaderboard_id uuid.UUID, if_modified_since time.Time) (LeaderboardResponseBody, *httptest.ResponseRecorder) {

	t.Helper()
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id),
		fmt.Sprintf("If-Modified-Since: %s", if_modified_since.UTC().Format(http.TimeFormat)))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getVerifiers(t *testing.T, api humatest.TestAPI, leaderboard_id uuid.UUID) (LeaderboardVerifiersResponseBody, *httptest.ResponseRecorder) {
	t.Helper()
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s/verifiers", leaderboard_id))
	var lResp LeaderboardVerifiersResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}
func getLeaderboardInfo(t *testing.T, api humatest.TestAPI, leaderboard_id uuid.UUID) (LeaderboardInfo, *httptest.ResponseRecorder) {
	t.Helper()
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s/info", leaderboard_id))
	var lResp LeaderboardInfo
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func getLeaderboard(t testing.TB, api humatest.TestAPI, leaderboard_id uuid.UUID) (LeaderboardResponseBody, *httptest.ResponseRecorder) {
	t.Helper()
	getResp := api.Get(fmt.Sprintf("/leaderboard/%s", leaderboard_id))
	var lResp LeaderboardResponseBody
	json.Unmarshal(getResp.Body.Bytes(), &lResp)
	return lResp, getResp
}

func TestLeaderboardInfoDefaults(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {

		display_name := "My Default Leaderboard"

		id := createDefaultLeaderboard(t, api, users["player2"])

		if lResp, getResp := getLeaderboardInfo(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, display_name, lResp.Title)
			assert.NotNil(t, lResp.Start)
			assert.Nil(t, lResp.Stop)
		}
	})
}

func TestLeaderboardInfo(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {
		display_name := "My First Leaderboard"

		id := createBasicLeaderboard(t, api, users["player2"])

		if lResp, getResp := getLeaderboardInfo(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, display_name, lResp.Title)
			assert.NotNil(t, lResp.Start)
			assert.NotNil(t, lResp.Stop)
		}
	})
}

func TestNewLeaderboard(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {

		id := createBasicLeaderboard(t, api, users["player2"])

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Zero(t, len(lResp.Scores))
		}

	})
}

func TestUnverifiedLeaderboard(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {
		id := createBasicLeaderboard(t, api, users["player2"])

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Zero(t, len(lResp.Scores))
		}

		postResp := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 9,
			})

		assert.Equal(t, 200, postResp.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 1, len(lResp.Scores))
			assert.Equal(t, 9, lResp.Scores[0].Score)
			assert.Nil(t, lResp.Scores[0].Verified)
		}

		postResp2 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 10,
			})

		assert.Equal(t, 200, postResp2.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 2, len(lResp.Scores))
			assert.Equal(t, 10, lResp.Scores[0].Score)
			assert.Equal(t, 9, lResp.Scores[1].Score)
			assert.Nil(t, lResp.Scores[0].Verified)
			assert.Nil(t, lResp.Scores[1].Verified)
		}

		postResp4 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 11,
			})

		assert.Equal(t, 200, postResp4.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 3, len(lResp.Scores))
			assert.Equal(t, 11, lResp.Scores[0].Score)
			assert.Equal(t, 10, lResp.Scores[1].Score)
			assert.Equal(t, 9, lResp.Scores[2].Score)

			assert.Nil(t, lResp.Scores[0].Verified)
			assert.Nil(t, lResp.Scores[1].Verified)
			assert.Nil(t, lResp.Scores[2].Verified)
		}
	})
}

func TestLeaderboardUpdateTrigger(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {
		id := createBasicLeaderboard(t, api, users["player2"])
		postResp2 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 10,
			})

		assert.Equal(t, 200, postResp2.Code)

		if _, getResp := getLeaderboardWithModifiedHeader(t, api, id, time.Now().AddDate(0, 1, 0)); assert.Equal(t, http.StatusNotModified, getResp.Code) {
			assert.NotEmpty(t, getResp.Header().Get("Last-Modified"))
		}
	})
}

func TestBadTimestamps(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {
		resp := api.Post("/leaderboard",
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"title":         "My First Leaderboard",
				"highest_first": true,
				"is_time":       true,
				"start":         time.Now().Format(time.RFC3339),
				"stop":          time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
				"verify":        true,
			})
		assert.Equal(t, 400, resp.Code)
	})
}

func TestVerifiedLeaderboard(t *testing.T) {
	WithApp(t, func(ctx context.Context, api humatest.TestAPI, users map[string]string) {
		id := createVerifiedLeaderboard(t, api, users["player2"])

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Zero(t, len(lResp.Scores))
		}

		postResp := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 9,
			})

		assert.Equal(t, 200, postResp.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 1, len(lResp.Scores))
			assert.Equal(t, 9, lResp.Scores[0].Score)
			assert.False(t, *lResp.Scores[0].Verified)
		}

		postResp2 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 10,
			})

		assert.Equal(t, 200, postResp2.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 2, len(lResp.Scores))
			assert.Equal(t, 10, lResp.Scores[0].Score)
			assert.Equal(t, 9, lResp.Scores[1].Score)
			assert.False(t, *lResp.Scores[0].Verified)
			assert.False(t, *lResp.Scores[1].Verified)
		}

		postResp4 := api.Post(
			fmt.Sprintf("/leaderboard/%s/submission", id),
			fmt.Sprintf("UserID: %s", users["player2"]),
			map[string]any{
				"link":  "www.youtube.com",
				"score": 11,
			})

		assert.Equal(t, 200, postResp4.Code)

		if lResp, getResp := getLeaderboard(t, api, id); assert.Equal(t, 200, getResp.Code) {
			assert.Equal(t, 3, len(lResp.Scores))
			assert.Equal(t, 11, lResp.Scores[0].Score)
			assert.Equal(t, 10, lResp.Scores[1].Score)
			assert.Equal(t, 9, lResp.Scores[2].Score)
			assert.False(t, *lResp.Scores[0].Verified)
			assert.False(t, *lResp.Scores[1].Verified)
			assert.False(t, *lResp.Scores[2].Verified)

		}
	})
}
