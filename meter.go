package main

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jackc/pgx/v5"
)

type CustomerInfo struct {
	id              int
	subscription_id int
	status          *string
}

const CUSTOMER_CONTEXT_KEY = "customer"

func (app *App) CustomerMiddleware(ctx huma.Context, next func(huma.Context)) {
	if user_id := ctx.Header("UserID"); len(user_id) > 0 {
		customer, db_err := app.st.getCustomer(ctx.Context(), user_id)
		if db_err != nil && db_err != pgx.ErrNoRows {
			huma.WriteErr(app.api, ctx, http.StatusInternalServerError,
				"Could not find user.", db_err,
			)
			return
		}

		ctx = huma.WithValue(ctx, CUSTOMER_CONTEXT_KEY, &customer)

		next(ctx)
	}
	// Set a custom header on the response.

	// Call the next middleware in the chain. This eventually calls the
	// operation handler as well.
}
