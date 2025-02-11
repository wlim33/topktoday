//go:build integration
// +build integration

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sqids/sqids-go"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() TestContext {
	app := App{
		db: setupDB(os.Getenv("DB_URL")),
	}

	s, _ := sqids.New(sqids.Options{
		MinLength: 9,
	})
	app.s = s
	router := app.setupRouter()
	return TestContext{
		router: router,
		w:      httptest.NewRecorder(),
	}

}

type TestContext struct {
	router *chi.Mux
	w      *httptest.ResponseRecorder
}

func encode(req interface{}) *bytes.Reader {
	jsonReq, _ := json.Marshal(req)
	return bytes.NewReader(jsonReq)
}

func (c *TestContext) jsonRequest(method string, path string, body io.Reader, target interface{}) error {
	req, _ := http.NewRequest(method, path, body)
	c.router.ServeHTTP(c.w, req)

	return json.NewDecoder(c.w.Body).Decode(&target)
}

func TestPingRoute(t *testing.T) {
	c := setupTestServer()

	var pongResponse MessageResponse
	c.jsonRequest("GET", "/ping", nil, &pongResponse)

	assert.Equal(t, 200, c.w.Code)
	assert.Equal(t, "pong", pongResponse.Message)

}

func TestAddScores(t *testing.T) {
	c := setupTestServer()

	var resp NewLeaderboardResponse
	c.jsonRequest("POST", "/leaderboard", encode(&NewLeaderboardRequest{
		DisplayName: "test name",
	}), &resp)
	assert.Equal(t, 200, c.w.Code)
	assert.Equal(t, "test name", resp.Name)

	var listResp LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &listResp)

	assert.Equal(t, 200, c.w.Code)
	assert.Zero(t, len(listResp.Scores))

	c.jsonRequest("POST",
		fmt.Sprintf("/leaderboard/%s/score", resp.Id),
		encode(UpdateScoreRequest{
			User:  "test",
			Score: 9,
		}),
		&MessageResponse{})
	assert.Equal(t, 200, c.w.Code)

	var secondListResp LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &secondListResp)

	assert.Equal(t, 200, c.w.Code)
	assert.Equal(t, 1, len(secondListResp.Scores))
	assert.Equal(t, 9, secondListResp.Scores[0].Score)

	c.jsonRequest("POST",
		fmt.Sprintf("/leaderboard/%s/score", resp.Id),
		encode(UpdateScoreRequest{
			User:  "test user 2",
			Score: 10,
		}),
		&MessageResponse{})
	assert.Equal(t, 200, c.w.Code)

	var thirdResp LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &thirdResp)
	assert.Equal(t, 10, thirdResp.Scores[0].Score)
	assert.Equal(t, 9, thirdResp.Scores[1].Score)

	c.jsonRequest("POST",
		fmt.Sprintf("/leaderboard/%s/score", resp.Id),
		encode(UpdateScoreRequest{
			User:  "test user 2",
			Score: 11,
		}),
		&MessageResponse{})
	assert.Equal(t, 200, c.w.Code)

	var fourth LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &fourth)
	assert.Equal(t, 2, len(fourth.Scores))
	assert.Equal(t, 11, fourth.Scores[0].Score)
	assert.Equal(t, 9, fourth.Scores[1].Score)

	c.jsonRequest("POST",
		fmt.Sprintf("/leaderboard/%s/score", resp.Id),
		encode(UpdateScoreRequest{
			User:  "test user 3",
			Score: 11,
		}),
		&MessageResponse{})
	assert.Equal(t, 200, c.w.Code)

	var fifth LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &fifth)
	assert.Equal(t, 3, len(fifth.Scores))
	assert.Equal(t, 11, fifth.Scores[0].Score)
	assert.Equal(t, 11, fifth.Scores[1].Score)
	assert.Equal(t, 9, fifth.Scores[2].Score)
}

func BenchmarkGetLeaderboard(b *testing.B) {
	c := setupTestServer()

	var resp NewLeaderboardResponse
	c.jsonRequest("POST", "/leaderboard", encode(&NewLeaderboardRequest{
		DisplayName: "test leaderboard",
	}), &resp)
	s, _ := sqids.New()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {

		id, _ := s.Encode([]uint64{uint64(n)})
		c.jsonRequest("POST",
			fmt.Sprintf("/leaderboard/%s/score", resp.Id),
			encode(UpdateScoreRequest{
				User:  id,
				Score: n,
			}),
			&MessageResponse{})

		var secondListResp LeaderboardResponse
		c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &secondListResp)
	}
}
