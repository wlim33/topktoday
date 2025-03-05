package main

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
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
	CustomerID      int    `json:"customer_id"`
	OrderID         int    `json:"order_id"`
	OrderItemID     int    `json:"order_item_id"`
	ProductID       int    `json:"product_id"`
	VariantID       int    `json:"variant_id"`
	CustomerName    string `json:"user_name"`
	CustomerEmail   string `json:"user_email"`
	Status          string `json:"status"`
	StatusFormatted string `json:"status_formatted"`
}

type WebhookBody struct {
	Meta struct {
		CustomData *WebhookCustomData `json:"custom_data"`
	} `json:"meta"`
	Data *struct {
		Attributes *CustomerAttributes `json:"attributes"`
	} `json:"data"`
}

type WebhookInput struct {
	Signature string `header:"X-Signature" required:"true"`
	EventName string `header:"X-Event-Name" required:"true"`
	RawBody   []byte
}

func (app *App) lemonPost(ctx context.Context, input *WebhookInput) (*WebhookResponse, error) {
	body := WebhookBody{}
	if err := json.Unmarshal(input.RawBody, &body); err != nil {
		log.Println("failed to unmarshal", err)
		return &WebhookResponse{Status: 422}, nil
	}
	app.webhookHash.Write(input.RawBody)

	if !hmac.Equal([]byte(hex.EncodeToString(app.webhookHash.Sum(nil))), []byte(input.Signature)) {
		log.Println("signature hash not valid")
		return &WebhookResponse{Status: 422}, nil
	}
	if body.Data == nil || body.Meta.CustomData == nil || body.Data.Attributes == nil {
		return &WebhookResponse{Status: 422}, nil
	}
	if err := app.st.updateSubscription(ctx, body.Meta.CustomData.UserID, body.Data.Attributes); err != nil {
		log.Println(err)
	}

	return &WebhookResponse{Status: 200}, nil
}
