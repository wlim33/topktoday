package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() TestContext {
	storage, err := NewStorage()
	if err != nil {
		log.Fatalln(err)
	}

	router := setupRouter(&storage)
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
	c.jsonRequest("POST", "/leaderboard", nil, &resp)
	assert.Equal(t, 200, c.w.Code)

	var listResp LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &listResp)

	assert.Equal(t, 200, c.w.Code)
	assert.Zero(t, len(listResp.Scores))

	c.jsonRequest("POST",
		fmt.Sprintf("/leaderboard/%s/score", resp.Id),
		encode(UpdateScoreRequest{
			User:  "test",
			Score: 10,
		}),
		&MessageResponse{})
	assert.Equal(t, 200, c.w.Code)

	var secondListResp LeaderboardResponse
	c.jsonRequest("GET", fmt.Sprintf("/leaderboard/%s", resp.Id), nil, &secondListResp)

	assert.Equal(t, 200, c.w.Code)
	assert.Equal(t, 1, len(secondListResp.Scores))
}
