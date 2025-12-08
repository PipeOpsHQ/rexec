/**
 * A/B Testing Utility for Rexec
 * 
 * Randomly assigns users to test variants and persists the assignment
 * so the same user sees the same variant on return visits.
 */

export type LandingVariant = 'original' | 'promo';

const AB_TEST_KEY = 'rexec_ab_landing';
const AB_TEST_VERSION = 'v1'; // Increment to reset all assignments

interface ABTestAssignment {
    variant: LandingVariant;
    version: string;
    assignedAt: number;
}

/**
 * Get or assign a landing page variant for the current user
 * Uses localStorage to persist assignment across sessions
 */
export function getLandingVariant(): LandingVariant {
    // Check for URL override (for testing)
    const params = new URLSearchParams(window.location.search);
    const override = params.get('landing');
    if (override === 'original' || override === 'promo') {
        return override;
    }

    // Check existing assignment
    try {
        const stored = localStorage.getItem(AB_TEST_KEY);
        if (stored) {
            const assignment: ABTestAssignment = JSON.parse(stored);
            // If same version, use stored variant
            if (assignment.version === AB_TEST_VERSION) {
                return assignment.variant;
            }
        }
    } catch {
        // Ignore parse errors
    }

    // Assign new variant (50/50 split)
    const variant: LandingVariant = Math.random() < 0.5 ? 'original' : 'promo';
    
    const assignment: ABTestAssignment = {
        variant,
        version: AB_TEST_VERSION,
        assignedAt: Date.now()
    };

    try {
        localStorage.setItem(AB_TEST_KEY, JSON.stringify(assignment));
    } catch {
        // localStorage might be full or disabled
    }

    return variant;
}

/**
 * Track an A/B test event (for analytics)
 */
export function trackABEvent(event: string, properties?: Record<string, unknown>): void {
    const variant = getLandingVariant();
    
    // If PostHog is available, track the event
    if (typeof window !== 'undefined' && (window as unknown as { posthog?: { capture: (event: string, props: Record<string, unknown>) => void } }).posthog) {
        (window as unknown as { posthog: { capture: (event: string, props: Record<string, unknown>) => void } }).posthog.capture(event, {
            ab_test: 'landing_page',
            ab_variant: variant,
            ab_version: AB_TEST_VERSION,
            ...properties
        });
    }
    
    // Log to console in development
    if (import.meta.env.DEV) {
        console.log(`[A/B Test] ${event}`, { variant, ...properties });
    }
}

/**
 * Force a specific variant (for testing/debugging)
 */
export function forceVariant(variant: LandingVariant): void {
    const assignment: ABTestAssignment = {
        variant,
        version: AB_TEST_VERSION,
        assignedAt: Date.now()
    };
    localStorage.setItem(AB_TEST_KEY, JSON.stringify(assignment));
}

/**
 * Clear A/B test assignment (user will get a new random assignment)
 */
export function clearAssignment(): void {
    localStorage.removeItem(AB_TEST_KEY);
}
