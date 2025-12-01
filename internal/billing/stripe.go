package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/stripe/stripe-go/v76"
	portalsession "github.com/stripe/stripe-go/v76/billingportal/session"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/customer"
	"github.com/stripe/stripe-go/v76/subscription"
	"github.com/stripe/stripe-go/v76/webhook"
)

var (
	ErrInvalidWebhookSignature = errors.New("invalid webhook signature")
	ErrCustomerNotFound        = errors.New("customer not found")
	ErrSubscriptionNotFound    = errors.New("subscription not found")
)

// Tier represents subscription tiers
type Tier string

const (
	TierFree       Tier = "free"
	TierPro        Tier = "pro"
	TierEnterprise Tier = "enterprise"
)

// SubscriptionStatus represents the status of a subscription
type SubscriptionStatus string

const (
	StatusActive   SubscriptionStatus = "active"
	StatusCanceled SubscriptionStatus = "canceled"
	StatusPastDue  SubscriptionStatus = "past_due"
	StatusTrialing SubscriptionStatus = "trialing"
	StatusInactive SubscriptionStatus = "inactive"
)

// CustomerInfo holds Stripe customer information
type CustomerInfo struct {
	CustomerID       string             `json:"customer_id"`
	SubscriptionID   string             `json:"subscription_id,omitempty"`
	Tier             Tier               `json:"tier"`
	Status           SubscriptionStatus `json:"status"`
	CurrentPeriodEnd time.Time          `json:"current_period_end,omitempty"`
}

// WebhookEvent represents a processed webhook event
type WebhookEvent struct {
	Type           string
	CustomerID     string
	SubscriptionID string
	Tier           Tier
	Status         SubscriptionStatus
}

// Config holds Stripe configuration
type Config struct {
	SecretKey         string
	WebhookSecret     string
	PriceIDPro        string
	PriceIDEnterprise string
	BaseURL           string
}

// Service handles Stripe billing operations
type Service struct {
	config Config
}

// NewService creates a new billing service
func NewService() *Service {
	config := Config{
		SecretKey:         os.Getenv("STRIPE_SECRET_KEY"),
		WebhookSecret:     os.Getenv("STRIPE_WEBHOOK_SECRET"),
		PriceIDPro:        os.Getenv("STRIPE_PRICE_PRO"),
		PriceIDEnterprise: os.Getenv("STRIPE_PRICE_ENTERPRISE"),
		BaseURL:           os.Getenv("BASE_URL"),
	}

	stripe.Key = config.SecretKey

	return &Service{
		config: config,
	}
}

// NewServiceWithConfig creates a billing service with explicit config
func NewServiceWithConfig(config Config) *Service {
	stripe.Key = config.SecretKey
	return &Service{config: config}
}

// CreateCustomer creates a new Stripe customer for a user
func (s *Service) CreateCustomer(ctx context.Context, userID, email, username string) (*stripe.Customer, error) {
	params := &stripe.CustomerParams{
		Email: stripe.String(email),
		Name:  stripe.String(username),
		Metadata: map[string]string{
			"user_id": userID,
		},
	}

	return customer.New(params)
}

// GetCustomer retrieves a Stripe customer by ID
func (s *Service) GetCustomer(ctx context.Context, customerID string) (*stripe.Customer, error) {
	return customer.Get(customerID, nil)
}

// CreateCheckoutSession creates a Stripe checkout session for subscription
func (s *Service) CreateCheckoutSession(ctx context.Context, customerID string, tier Tier) (*stripe.CheckoutSession, error) {
	var priceID string
	switch tier {
	case TierPro:
		priceID = s.config.PriceIDPro
	case TierEnterprise:
		priceID = s.config.PriceIDEnterprise
	default:
		return nil, fmt.Errorf("invalid tier for checkout: %s", tier)
	}

	if priceID == "" {
		return nil, fmt.Errorf("price ID not configured for tier: %s", tier)
	}

	params := &stripe.CheckoutSessionParams{
		Customer: stripe.String(customerID),
		Mode:     stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.config.BaseURL + "/billing/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.config.BaseURL + "/billing/cancel"),
		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"tier": string(tier),
			},
		},
	}

	return session.New(params)
}

// CreateBillingPortalSession creates a session for the Stripe billing portal
func (s *Service) CreateBillingPortalSession(ctx context.Context, customerID string) (string, error) {
	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(customerID),
		ReturnURL: stripe.String(s.config.BaseURL + "/dashboard"),
	}

	sess, err := portalsession.New(params)
	if err != nil {
		return "", fmt.Errorf("failed to create portal session: %w", err)
	}

	return sess.URL, nil
}

// GetSubscription retrieves a subscription by ID
func (s *Service) GetSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	return subscription.Get(subscriptionID, nil)
}

// CancelSubscription cancels a subscription at period end
func (s *Service) CancelSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}
	return subscription.Update(subscriptionID, params)
}

// ReactivateSubscription reactivates a subscription that was set to cancel
func (s *Service) ReactivateSubscription(ctx context.Context, subscriptionID string) (*stripe.Subscription, error) {
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(false),
	}
	return subscription.Update(subscriptionID, params)
}

// ChangeSubscriptionTier changes the subscription to a different tier
func (s *Service) ChangeSubscriptionTier(ctx context.Context, subscriptionID string, newTier Tier) (*stripe.Subscription, error) {
	var priceID string
	switch newTier {
	case TierPro:
		priceID = s.config.PriceIDPro
	case TierEnterprise:
		priceID = s.config.PriceIDEnterprise
	default:
		return nil, fmt.Errorf("invalid tier: %s", newTier)
	}

	// Get current subscription to find the item ID
	sub, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	if len(sub.Items.Data) == 0 {
		return nil, errors.New("subscription has no items")
	}

	itemID := sub.Items.Data[0].ID

	params := &stripe.SubscriptionParams{
		Items: []*stripe.SubscriptionItemsParams{
			{
				ID:    stripe.String(itemID),
				Price: stripe.String(priceID),
			},
		},
		ProrationBehavior: stripe.String(string(stripe.SubscriptionSchedulePhaseProrationBehaviorCreateProrations)),
		Metadata: map[string]string{
			"tier": string(newTier),
		},
	}

	return subscription.Update(subscriptionID, params)
}

// GetCustomerSubscriptionInfo gets the current subscription info for a customer
func (s *Service) GetCustomerSubscriptionInfo(ctx context.Context, customerID string) (*CustomerInfo, error) {
	info := &CustomerInfo{
		CustomerID: customerID,
		Tier:       TierFree,
		Status:     StatusInactive,
	}

	// List active subscriptions for customer
	params := &stripe.SubscriptionListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String(string(stripe.SubscriptionStatusActive)),
	}
	params.Limit = stripe.Int64(1)

	iter := subscription.List(params)
	if iter.Next() {
		sub := iter.Subscription()
		info.SubscriptionID = sub.ID
		info.Status = mapSubscriptionStatus(sub.Status)
		info.CurrentPeriodEnd = time.Unix(sub.CurrentPeriodEnd, 0)

		// Get tier from metadata or price
		if tier, ok := sub.Metadata["tier"]; ok {
			info.Tier = Tier(tier)
		} else if len(sub.Items.Data) > 0 {
			info.Tier = s.getTierFromPriceID(sub.Items.Data[0].Price.ID)
		}
	}

	if err := iter.Err(); err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	return info, nil
}

// HandleWebhook processes a Stripe webhook event
func (s *Service) HandleWebhook(r *http.Request) (*WebhookEvent, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}

	signature := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(body, signature, s.config.WebhookSecret)
	if err != nil {
		return nil, ErrInvalidWebhookSignature
	}

	return s.processWebhookEvent(&event)
}

// processWebhookEvent processes different webhook event types
func (s *Service) processWebhookEvent(event *stripe.Event) (*WebhookEvent, error) {
	result := &WebhookEvent{
		Type: string(event.Type),
	}

	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":

		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return nil, fmt.Errorf("failed to parse subscription: %w", err)
		}

		result.CustomerID = sub.Customer.ID
		result.SubscriptionID = sub.ID
		result.Status = mapSubscriptionStatus(sub.Status)

		if tier, ok := sub.Metadata["tier"]; ok {
			result.Tier = Tier(tier)
		} else if len(sub.Items.Data) > 0 {
			result.Tier = s.getTierFromPriceID(sub.Items.Data[0].Price.ID)
		}

		// Handle deleted subscription
		if event.Type == "customer.subscription.deleted" {
			result.Tier = TierFree
			result.Status = StatusCanceled
		}

	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return nil, fmt.Errorf("failed to parse checkout session: %w", err)
		}

		result.CustomerID = sess.Customer.ID
		result.SubscriptionID = sess.Subscription.ID
		result.Status = StatusActive

	case "invoice.payment_failed":
		var invoice stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &invoice); err != nil {
			return nil, fmt.Errorf("failed to parse invoice: %w", err)
		}

		result.CustomerID = invoice.Customer.ID
		if invoice.Subscription != nil {
			result.SubscriptionID = invoice.Subscription.ID
		}
		result.Status = StatusPastDue

	default:
		// Unhandled event type
		return nil, nil
	}

	return result, nil
}

// getTierFromPriceID determines the tier from a Stripe price ID
func (s *Service) getTierFromPriceID(priceID string) Tier {
	switch priceID {
	case s.config.PriceIDPro:
		return TierPro
	case s.config.PriceIDEnterprise:
		return TierEnterprise
	default:
		return TierFree
	}
}

// mapSubscriptionStatus maps Stripe subscription status to our status
func mapSubscriptionStatus(status stripe.SubscriptionStatus) SubscriptionStatus {
	switch status {
	case stripe.SubscriptionStatusActive:
		return StatusActive
	case stripe.SubscriptionStatusCanceled:
		return StatusCanceled
	case stripe.SubscriptionStatusPastDue:
		return StatusPastDue
	case stripe.SubscriptionStatusTrialing:
		return StatusTrialing
	default:
		return StatusInactive
	}
}

// TierLimits returns container limits for each tier
func TierLimits(tier Tier) int {
	switch tier {
	case TierPro:
		return 5
	case TierEnterprise:
		return 20
	default:
		return 2
	}
}
