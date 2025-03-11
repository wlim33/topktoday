//go:build integration
// +build integration

package main

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
)

func TestNewLeaderboardDB(t *testing.T) {
	WithTx(t, func(ctx context.Context, tx pgx.Tx) {

		db := DB{
			conn: tx,
		}
		err := db.createTestUser(ctx, "meowid", "meow", "meow@meow", false, CustomerInfo{
			id:              123123,
			subscription_id: 123123,
		})
		assert.NoError(t, err)

		leaderboard_id, err := db.newLeaderboard(ctx, "meowid", LeaderboardConfig{
			Title: "My Leaderboard",
		})

		assert.NotEqual(t, uuid.Nil, leaderboard_id)
		assert.NoError(t, err)
	})
}

func TestCreateTestUser(t *testing.T) {
	WithTx(t, func(ctx context.Context, tx pgx.Tx) {

		db := DB{
			conn: tx,
		}

		err := db.createTestUser(ctx, "meowid", "meow", "meow@meow", false, CustomerInfo{
			id:              123123,
			subscription_id: 123123,
		})
		if err != nil {
			assert.Fail(t, "oiaernt")
		}
		err = db.createTestUser(ctx, "meowid", "meow", "meow@meow", false, CustomerInfo{
			id:              123123,
			subscription_id: 123123,
		})
		if err != nil {
			assert.Fail(t, "oiaernt")
		}

		err = db.createTestUser(ctx, "meowid", "meow", "meow@meow", false, CustomerInfo{
			id:              123123,
			subscription_id: 123123,
		})
		if err != nil {
			assert.Fail(t, "oiaernt")
		}

		info, g_err := db.getCustomer(ctx, "meowid")

		if g_err != nil {
			assert.Fail(t, "oiaernt")
		}
		assert.Equal(t, 123123, info.id)
		assert.NoError(t, err)
	})
}
