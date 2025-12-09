package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rexec/rexec/internal/billing"
	"github.com/rexec/rexec/internal/models"
	"github.com/rexec/rexec/internal/storage"
)

// BillingHandler handles billing-related API endpoints
type BillingHandler struct {
	billingService *billing.Service
	store          *storage.PostgresStore
}

// NewBillingHandler creates a new billing handler
func NewBillingHandler(billingService *billing.Service, store *storage.PostgresStore) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
		store:          store,
	}
}

// CreateCheckoutSessionRequest represents the request to create a checkout session
type CreateCheckoutSessionRequest struct {
	Tier string `json:"tier" binding:"required,oneof=pro enterprise"`
}

// CreateCheckoutSession creates a Stripe checkout session for subscription upgrade
// POST /api/billing/checkout
func (h *BillingHandler) CreateCheckoutSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateCheckoutSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	// Get user to retrieve or create Stripe customer
	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get or create Stripe customer ID from user metadata
	customerID, err := h.getOrCreateCustomerID(c, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create customer: " + err.Error()})
		return
	}

	// Create checkout session
	session, err := h.billingService.CreateCheckoutSession(
		c.Request.Context(),
		customerID,
		billing.Tier(req.Tier),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create checkout session: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checkout_url": session.URL,
		"session_id":   session.ID,
	})
}

// GetSubscription returns the current subscription status
// GET /api/billing/subscription
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Get customer ID from user metadata
	customerID, err := h.store.GetUserStripeCustomerID(c.Request.Context(), user.ID)
	if err != nil || customerID == "" {
		// No Stripe customer, return free tier info
		c.JSON(http.StatusOK, gin.H{
			"tier":            "free",
			"status":          "active",
			"container_limit": billing.TierLimits(billing.TierFree),
		})
		return
	}

	// Get subscription info from Stripe
	info, err := h.billingService.GetCustomerSubscriptionInfo(c.Request.Context(), customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get subscription info"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tier":               info.Tier,
		"status":             info.Status,
		"container_limit":    billing.TierLimits(info.Tier),
		"current_period_end": info.CurrentPeriodEnd,
	})
}

// CreatePortalSession creates a Stripe billing portal session
// POST /api/billing/portal
func (h *BillingHandler) CreatePortalSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	customerID, err := h.store.GetUserStripeCustomerID(c.Request.Context(), user.ID)
	if err != nil || customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active subscription"})
		return
	}

	portalURL, err := h.billingService.CreateBillingPortalSession(c.Request.Context(), customerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create portal session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"portal_url": portalURL,
	})
}

// CancelSubscription cancels the current subscription at period end
// POST /api/billing/cancel
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	customerID, err := h.store.GetUserStripeCustomerID(c.Request.Context(), user.ID)
	if err != nil || customerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active subscription"})
		return
	}

	// Get current subscription
	info, err := h.billingService.GetCustomerSubscriptionInfo(c.Request.Context(), customerID)
	if err != nil || info.SubscriptionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no active subscription"})
		return
	}

	// Cancel at period end
	_, err = h.billingService.CancelSubscription(c.Request.Context(), info.SubscriptionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel subscription"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":            "subscription will be canceled at period end",
		"current_period_end": info.CurrentPeriodEnd,
	})
}

// HandleWebhook handles Stripe webhook events
// POST /api/billing/webhook
func (h *BillingHandler) HandleWebhook(c *gin.Context) {
	event, err := h.billingService.HandleWebhook(c.Request)
	if err != nil {
		if err == billing.ErrInvalidWebhookSignature {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Event was not one we handle
	if event == nil {
		c.JSON(http.StatusOK, gin.H{"received": true})
		return
	}

	// Process the event
	ctx := c.Request.Context()

	switch event.Type {
	case "customer.subscription.created",
		"customer.subscription.updated",
		"checkout.session.completed":

		// Update user tier based on subscription
		userID, err := h.store.GetUserIDByStripeCustomerID(ctx, event.CustomerID)
		if err != nil || userID == "" {
			// Customer not linked to a user yet, may happen on checkout completion
			c.JSON(http.StatusOK, gin.H{"received": true, "note": "customer not linked"})
			return
		}

		// Update user's tier
		if err := h.store.UpdateUserTier(ctx, userID, string(event.Tier)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user tier"})
			return
		}

	case "customer.subscription.deleted":
		// Downgrade to free tier
		userID, err := h.store.GetUserIDByStripeCustomerID(ctx, event.CustomerID)
		if err != nil || userID == "" {
			c.JSON(http.StatusOK, gin.H{"received": true})
			return
		}

		if err := h.store.UpdateUserTier(ctx, userID, string(billing.TierFree)); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user tier"})
			return
		}

	case "invoice.payment_failed":
		// Could send notification to user, mark account, etc.
		// For now, just log and acknowledge
	}

	c.JSON(http.StatusOK, gin.H{"received": true})
}

// getOrCreateCustomerID retrieves existing or creates new Stripe customer
func (h *BillingHandler) getOrCreateCustomerID(c *gin.Context, user *models.User) (string, error) {
	ctx := c.Request.Context()

	// Check if user already has a Stripe customer ID
	customerID, err := h.store.GetUserStripeCustomerID(ctx, user.ID)
	if err != nil {
		return "", err
	}

	if customerID != "" {
		return customerID, nil
	}

	// Create new Stripe customer
	customer, err := h.billingService.CreateCustomer(ctx, user.ID, user.Email, user.Username)
	if err != nil {
		return "", err
	}

	// Save customer ID to database
	if err := h.store.SetUserStripeCustomerID(ctx, user.ID, customer.ID); err != nil {
		return "", err
	}

	return customer.ID, nil
}

// GetPlans returns available subscription plans
// GET /api/billing/plans
func (h *BillingHandler) GetPlans(c *gin.Context) {
	plans := []gin.H{
		{
			"tier":            "free",
			"name":            "Free",
			"price":           0,
			"container_limit": billing.TierLimits(billing.TierFree),
			"features": []string{
				"2 containers",
				"512 MB RAM per container",
				"1 GB storage",
				"Community support",
			},
		},
		{
			"tier":            "pro",
			"name":            "Pro",
			"price":           9.99,
			"container_limit": billing.TierLimits(billing.TierPro),
			"features": []string{
				"5 containers",
				"2 GB RAM per container",
				"10 GB storage",
				"Priority support",
				"Custom images",
			},
		},
		{
			"tier":            "enterprise",
			"name":            "Enterprise",
			"price":           29.99,
			"container_limit": billing.TierLimits(billing.TierEnterprise),
			"features": []string{
				"20 containers",
				"4 GB RAM per container",
				"50 GB storage",
				"Dedicated support",
				"Custom images",
				"SSH access",
				"Team collaboration",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{"plans": plans})
}

// GetBillingHistory returns the user's invoice history
// GET /api/billing/history
func (h *BillingHandler) GetBillingHistory(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get user"})
		return
	}
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	customerID, err := h.store.GetUserStripeCustomerID(c.Request.Context(), user.ID)
	if err != nil || customerID == "" {
		// No Stripe customer, return empty history
		c.JSON(http.StatusOK, gin.H{"invoices": []interface{}{}})
		return
	}

	invoices, err := h.billingService.ListInvoices(c.Request.Context(), customerID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get billing history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}
