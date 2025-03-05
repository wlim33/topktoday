//go:build integration
// +build integration

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const webhook_payload = `
{
  "meta": {
    "event_name": "order_created",
    "custom_data": {
      "user_id": "testid"
    }
  },
  "data": {
    "type": "orders",
    "id": "1",
    "attributes": {
      "store_id": 1,
      "customer_id": 1,
      "identifier": "104e18a2-d755-4d4b-80c4-a6c1dcbe1c10",
      "order_number": 1,
      "user_name": "John Doe",
      "user_email": "johndoe@example.com",
      "currency": "USD",
      "currency_rate": "1.0000",
      "subtotal": 999,
      "discount_total": 0,
      "tax": 200,
      "total": 1199,
      "subtotal_usd": 999,
      "discount_total_usd": 0,
      "tax_usd": 200,
      "total_usd": 1199,
      "tax_name": "VAT",
      "tax_rate": "20.00",
      "status": "paid",
      "status_formatted": "Paid",
      "refunded": false,
      "refunded_at": null,
      "subtotal_formatted": "$9.99",
      "discount_total_formatted": "$0.00",
      "tax_formatted": "$2.00",
      "total_formatted": "$11.99",
      "first_order_item": {
        "id": 1,
        "order_id": 1,
        "product_id": 1,
        "variant_id": 1,
        "product_name": "Test Limited License for 2 years",
        "variant_name": "Default",
        "price": 1199,
        "created_at": "2021-08-17T09:45:53.000000Z",
        "updated_at": "2021-08-17T09:45:53.000000Z",
        "deleted_at": null,
        "test_mode": false
      },
      "urls": {
        "receipt": "https://app.lemonsqueezy.com/my-orders/104e18a2-d755-4d4b-80c4-a6c1dcbe1c10?signature=8847fff02e1bfb0c7c43ff1cdf1b1657a8eed2029413692663b86859208c9f42"
      },
      "created_at": "2021-08-17T09:45:53.000000Z",
      "updated_at": "2021-08-17T09:45:53.000000Z"
    },
    "relationships": {
      "store": {
        "links": {
          "related": "https://api.lemonsqueezy.com/v1/orders/1/store",
          "self": "https://api.lemonsqueezy.com/v1/orders/1/relationships/store"
        }
      },
      "customer": {
        "links": {
          "related": "https://api.lemonsqueezy.com/v1/orders/1/customer",
          "self": "https://api.lemonsqueezy.com/v1/orders/1/relationships/customer"
        }
      },
      "order-items": {
        "links": {
          "related": "https://api.lemonsqueezy.com/v1/orders/1/order-items",
          "self": "https://api.lemonsqueezy.com/v1/orders/1/relationships/order-items"
        }
      },
      "subscriptions": {
        "links": {
          "related": "https://api.lemonsqueezy.com/v1/orders/1/subscriptions",
          "self": "https://api.lemonsqueezy.com/v1/orders/1/relationships/subscriptions"
        }
      },
      "license-keys": {
        "links": {
          "related": "https://api.lemonsqueezy.com/v1/orders/1/license-keys",
          "self": "https://api.lemonsqueezy.com/v1/orders/1/relationships/license-keys"
        }
      },
      "discount-redemptions": {
        "links": {
          "related": "https://api.lemonsqueezy.com/v1/orders/1/discount-redemptions",
          "self": "https://api.lemonsqueezy.com/v1/orders/1/relationships/discount-redemptions"
        }
      }
    },
    "links": {
      "self": "https://api.lemonsqueezy.com/v1/orders/1"
    }
  }
}
	`

func TestWebhook(t *testing.T) {
	key := "test_signing_key"
	api := setupTestWebhookAPI(t, key)

	hash := hmac.New(sha256.New, []byte(key))
	hash.Write([]byte(webhook_payload))
	resp := api.Post("/webhooks/lemon_squeezy",
		"Content-Type: application/json",
		"X-Event-Name: order_created",
		fmt.Sprintf("X-Signature: %s", hash.Sum(nil)),
		strings.NewReader(webhook_payload))

	assert.Equal(t, 200, resp.Code)
}
