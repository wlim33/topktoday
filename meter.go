package main

import (
	"bytes"
	"context"
	"fmt"
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

	}

	next(ctx)
}

func (app *App) getUsage(ctx context.Context, subscription_id int) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.lemonsqueezy.com/v1/subscription-items/%d/current-usage", subscription_id), nil)

	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", app.lsApiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (app *App) sendUsageUpdate(subscription_id int, amount int) error {
	body := []byte(fmt.Sprintf(`{
    "data": {
      "type": "usage-records",
      "attributes": {
        "quantity": %d
      },
	  "relationships": {
		"subscription-item": {
		  "data": {
            "type": "subscription-items",
            "id": "%d"
          }		
		}
	  }
    }
  }`, amount, subscription_id))
	req, err := http.NewRequest(http.MethodPost, "https://api.lemonsqueezy.com/v1/usage-records", bytes.NewBuffer(body))

	req.Header.Set("Accept", "application/vnd.api+json")
	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", app.lsApiKey))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
