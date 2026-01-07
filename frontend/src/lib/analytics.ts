import posthog from "posthog-js";

/**
 * PostHog Analytics Utilities
 *
 * Use these functions throughout the app to track custom events
 * and identify users.
 */

/**
 * Identify a user after login/signup
 */
export function identifyUser(
  userId: string,
  properties?: {
    email?: string;
    name?: string;
    tier?: string;
    [key: string]: unknown;
  },
) {
  posthog.identify(userId, properties);
}

/**
 * Reset user identity on logout
 */
export function resetUser() {
  posthog.reset();
}

/**
 * Track a custom event
 */
export function trackEvent(
  eventName: string,
  properties?: Record<string, unknown>,
) {
  posthog.capture(eventName, properties);
}

/**
 * Track container creation
 */
export function trackContainerCreated(properties: {
  image: string;
  role?: string;
  isAgent?: boolean;
}) {
  trackEvent("container_created", properties);
}

/**
 * Track container connection
 */
export function trackContainerConnected(properties: {
  containerId: string;
  image?: string;
}) {
  trackEvent("container_connected", properties);
}

/**
 * Track collaboration session started
 */
export function trackCollabSessionStarted(properties: {
  mode: "view" | "control";
  maxUsers: number;
}) {
  trackEvent("collab_session_started", properties);
}

/**
 * Track feature usage
 */
export function trackFeatureUsed(
  feature: string,
  properties?: Record<string, unknown>,
) {
  trackEvent("feature_used", { feature, ...properties });
}

/**
 * Set user properties without triggering an event
 */
export function setUserProperties(properties: Record<string, unknown>) {
  posthog.people.set(properties);
}

/**
 * Track page view (useful for SPA navigation)
 */
export function trackPageView(path?: string) {
  posthog.capture("$pageview", {
    $current_url: path || window.location.href,
  });
}

// Re-export posthog for advanced usage
export { posthog };
