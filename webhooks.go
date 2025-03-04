package main

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"log"
)

type WebhookResponse struct {
	Status int
}
type WebhookCustomData struct {
	UserID string `json:"user_id"`
}
type CustomerAttributes struct {
	CustomerID      json.Number `json:"customer_id"`
	OrderID         json.Number `json:"order_id"`
	OrderItemID     json.Number `json:"order_item_id"`
	ProductID       json.Number `json:"product_id"`
	VariantID       json.Number `json:"variant_id"`
	CustomerName    string      `json:"user_name"`
	CustomerEmail   string      `json:"user_email"`
	Status          string      `json:"status"`
	StatusFormatted string      `json:"status_formatted"`
}

type WebhookInput struct {
	Signature string `header:"X-Signature" required:"true"`
	EventName string `header:"X-Event-Name" required:"true"`
	Body      struct {
		CustomData *WebhookCustomData `json:"custom_data"`
		Data       *struct {
			Attributes *CustomerAttributes `json:"attributes"`
		} `json:"data"`
	}
	RawBody []byte
}

func (app *App) lemonPost(ctx context.Context, input *WebhookInput) (*WebhookResponse, error) {
	if !hmac.Equal(app.webhookHash.Sum(input.RawBody), []byte(input.Signature)) {
		return &WebhookResponse{Status: 500}, nil
	}
	err := app.st.updateSubscription(ctx, input.Body.CustomData.UserID, input.Body.Data.Attributes)
	if err != nil {
		log.Println(err)
	}

	return &WebhookResponse{Status: 200}, nil
}
