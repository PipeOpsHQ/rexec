<script lang="ts">
    import { onMount, createEventDispatcher } from "svelte";
    import { billing, type Invoice, type Subscription } from "$stores/billing";
    import { userTier } from "$stores/auth";
    import { toast } from "$stores/toast";
    import StatusIcon from "./icons/StatusIcon.svelte";

    const dispatch = createEventDispatcher<{
        back: void;
        pricing: void;
    }>();

    let subscription: Subscription | null = null;
    let invoices: Invoice[] = [];
    let isLoading = true;
    let isPortalLoading = false;

    onMount(async () => {
        isLoading = true;
        const [sub] = await Promise.all([
            billing.fetchSubscription(),
            billing.fetchInvoices(),
        ]);
        subscription = sub;
        
        billing.subscribe((state) => {
            invoices = state.invoices;
        });
        isLoading = false;
    });

    function formatCurrency(amount: number, currency: string): string {
        return new Intl.NumberFormat("en-US", {
            style: "currency",
            currency: currency.toUpperCase(),
        }).format(amount / 100);
    }

    function formatDate(dateStr: string): string {
        return new Date(dateStr).toLocaleDateString("en-US", {
            year: "numeric",
            month: "short",
            day: "numeric",
        });
    }

    function getStatusColor(status: string): string {
        switch (status) {
            case "paid":
                return "var(--accent)";
            case "open":
            case "draft":
                return "var(--warning)";
            case "void":
            case "uncollectible":
                return "var(--error)";
            default:
                return "var(--text-muted)";
        }
    }

    function getTierBadgeColor(tier: string): string {
        switch (tier) {
            case "pro":
                return "#00d4ff";
            case "enterprise":
                return "#8b5cf6";
            default:
                return "var(--text-muted)";
        }
    }

    async function openBillingPortal() {
        isPortalLoading = true;
        const result = await billing.openPortal();
        isPortalLoading = false;

        if (result.success && result.portalUrl) {
            window.open(result.portalUrl, "_blank");
        } else {
            toast.error(result.error || "Failed to open billing portal");
        }
    }

    async function handleUpgrade(tier: "pro" | "enterprise") {
        const result = await billing.createCheckout(tier);
        if (result.success && result.checkoutUrl) {
            window.location.href = result.checkoutUrl;
        } else {
            toast.error(result.error || "Failed to start checkout");
        }
    }
</script>

<div class="billing-page">
    <div class="page-header">
        <button class="back-btn" on:click={() => dispatch("back")}>
            <StatusIcon status="back" size={16} />
            Back
        </button>
        <h1>Billing</h1>
    </div>

    {#if isLoading}
        <div class="loading">
            <div class="spinner"></div>
            <p>Loading billing information...</p>
        </div>
    {:else}
        <!-- Current Plan Section -->
        <section class="section">
            <h2>Current Plan</h2>
            <div class="plan-card">
                <div class="plan-info">
                    <div class="plan-header">
                        <span
                            class="tier-badge"
                            style="background: {getTierBadgeColor($userTier || 'free')}20; color: {getTierBadgeColor($userTier || 'free')}"
                        >
                            {($userTier || "free").toUpperCase()}
                        </span>
                        {#if subscription?.status === "active"}
                            <span class="status-badge active">Active</span>
                        {:else if subscription?.status === "past_due"}
                            <span class="status-badge past-due">Past Due</span>
                        {:else if subscription?.status === "canceled"}
                            <span class="status-badge canceled">Canceled</span>
                        {/if}
                    </div>

                    {#if subscription?.current_period_end}
                        <p class="period-info">
                            {#if subscription.status === "canceled"}
                                Access until {formatDate(subscription.current_period_end)}
                            {:else}
                                Next billing date: {formatDate(subscription.current_period_end)}
                            {/if}
                        </p>
                    {/if}

                    <p class="limit-info">
                        Container limit: <strong>{subscription?.container_limit || 2}</strong>
                    </p>
                </div>

                <div class="plan-actions">
                    {#if $userTier === "free" || $userTier === "guest"}
                        <button class="btn btn-primary" on:click={() => dispatch("pricing")}>
                            Upgrade Plan
                        </button>
                    {:else}
                        <button
                            class="btn btn-secondary"
                            on:click={openBillingPortal}
                            disabled={isPortalLoading}
                        >
                            {#if isPortalLoading}
                                <span class="spinner-small"></span>
                            {:else}
                                Manage Subscription
                            {/if}
                        </button>
                    {/if}
                </div>
            </div>
        </section>

        <!-- Billing History Section -->
        <section class="section">
            <h2>Billing History</h2>

            {#if invoices.length === 0}
                <div class="empty-state">
                    <StatusIcon status="invoice" size={48} />
                    <p>No billing history yet</p>
                    <span class="empty-hint">
                        Your invoices will appear here once you subscribe to a paid plan.
                    </span>
                </div>
            {:else}
                <div class="invoices-table">
                    <div class="table-header">
                        <span class="col-date">Date</span>
                        <span class="col-number">Invoice</span>
                        <span class="col-amount">Amount</span>
                        <span class="col-status">Status</span>
                        <span class="col-actions">Actions</span>
                    </div>

                    {#each invoices as invoice}
                        <div class="table-row">
                            <span class="col-date">{formatDate(invoice.created)}</span>
                            <span class="col-number">{invoice.number || "—"}</span>
                            <span class="col-amount">
                                {formatCurrency(invoice.amount_paid || invoice.amount_due, invoice.currency)}
                            </span>
                            <span class="col-status">
                                <span
                                    class="status-pill"
                                    style="color: {getStatusColor(invoice.status)}"
                                >
                                    {invoice.status}
                                </span>
                            </span>
                            <span class="col-actions">
                                {#if invoice.hosted_invoice_url}
                                    <a
                                        href={invoice.hosted_invoice_url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="action-link"
                                    >
                                        View
                                    </a>
                                {/if}
                                {#if invoice.invoice_pdf}
                                    <a
                                        href={invoice.invoice_pdf}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class="action-link"
                                    >
                                        PDF
                                    </a>
                                {/if}
                            </span>
                        </div>
                    {/each}
                </div>
            {/if}
        </section>

        <!-- Payment Methods Section -->
        <section class="section">
            <h2>Payment Methods</h2>
            <div class="payment-info">
                <p>
                    Payment methods are managed through our secure billing portal powered by Stripe.
                </p>
                {#if $userTier !== "free" && $userTier !== "guest"}
                    <button
                        class="btn btn-secondary"
                        on:click={openBillingPortal}
                        disabled={isPortalLoading}
                    >
                        {#if isPortalLoading}
                            <span class="spinner-small"></span>
                        {:else}
                            Manage Payment Methods
                        {/if}
                    </button>
                {/if}
            </div>
        </section>
    {/if}
</div>

<style>
    .billing-page {
        max-width: 900px;
        margin: 0 auto;
        padding: 20px;
    }

    .page-header {
        display: flex;
        align-items: center;
        gap: 16px;
        margin-bottom: 32px;
    }

    .page-header h1 {
        font-size: 24px;
        font-weight: 600;
        margin: 0;
    }

    .back-btn {
        display: flex;
        align-items: center;
        gap: 6px;
        background: none;
        border: none;
        color: var(--text-secondary);
        cursor: pointer;
        font-size: 14px;
        padding: 8px 12px;
        border-radius: 6px;
        transition: all 0.2s;
    }

    .back-btn:hover {
        background: var(--bg-secondary);
        color: var(--text);
    }

    .loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 80px 20px;
        gap: 16px;
    }

    .loading p {
        color: var(--text-muted);
    }

    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    .spinner-small {
        width: 16px;
        height: 16px;
        border: 2px solid var(--border);
        border-top-color: var(--accent);
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
        to { transform: rotate(360deg); }
    }

    .section {
        margin-bottom: 40px;
    }

    .section h2 {
        font-size: 16px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-secondary);
        margin-bottom: 16px;
    }

    .plan-card {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 24px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 20px;
    }

    .plan-header {
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 12px;
    }

    .tier-badge {
        padding: 4px 12px;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 600;
        letter-spacing: 0.5px;
    }

    .status-badge {
        padding: 4px 10px;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 500;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }

    .status-badge.active {
        background: rgba(0, 255, 136, 0.15);
        color: var(--accent);
    }

    .status-badge.past-due {
        background: rgba(255, 193, 7, 0.15);
        color: var(--warning);
    }

    .status-badge.canceled {
        background: rgba(255, 107, 107, 0.15);
        color: var(--error);
    }

    .period-info,
    .limit-info {
        font-size: 14px;
        color: var(--text-secondary);
        margin: 4px 0;
    }

    .limit-info strong {
        color: var(--text);
    }

    .plan-actions {
        display: flex;
        gap: 12px;
    }

    .btn {
        padding: 10px 20px;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        display: flex;
        align-items: center;
        gap: 8px;
    }

    .btn-primary {
        background: var(--accent);
        color: #000;
        border: none;
    }

    .btn-primary:hover {
        background: var(--accent-hover);
        transform: translateY(-1px);
    }

    .btn-secondary {
        background: var(--bg-secondary);
        color: var(--text);
        border: 1px solid var(--border);
    }

    .btn-secondary:hover {
        border-color: var(--accent);
        background: var(--bg-card);
    }

    .btn:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 60px 20px;
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        color: var(--text-muted);
    }

    .empty-state p {
        margin-top: 16px;
        font-size: 16px;
        color: var(--text-secondary);
    }

    .empty-hint {
        font-size: 13px;
        margin-top: 8px;
    }

    .invoices-table {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        overflow: hidden;
    }

    .table-header,
    .table-row {
        display: grid;
        grid-template-columns: 120px 1fr 100px 100px 100px;
        align-items: center;
        padding: 14px 20px;
        gap: 12px;
    }

    .table-header {
        background: var(--bg-secondary);
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
        letter-spacing: 0.5px;
        color: var(--text-muted);
    }

    .table-row {
        border-top: 1px solid var(--border);
        font-size: 14px;
    }

    .table-row:hover {
        background: var(--bg-secondary);
    }

    .status-pill {
        font-size: 12px;
        text-transform: capitalize;
    }

    .col-actions {
        display: flex;
        gap: 12px;
    }

    .action-link {
        color: var(--accent);
        text-decoration: none;
        font-size: 13px;
        transition: opacity 0.2s;
    }

    .action-link:hover {
        opacity: 0.8;
    }

    .payment-info {
        background: var(--bg-card);
        border: 1px solid var(--border);
        border-radius: 8px;
        padding: 24px;
    }

    .payment-info p {
        font-size: 14px;
        color: var(--text-secondary);
        margin-bottom: 16px;
    }

    @media (max-width: 768px) {
        .plan-card {
            flex-direction: column;
            align-items: flex-start;
        }

        .table-header,
        .table-row {
            grid-template-columns: 1fr 1fr;
            gap: 8px;
        }

        .col-number,
        .col-status {
            display: none;
        }
    }
</style>
