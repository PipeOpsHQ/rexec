<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { userTier } from "$stores/auth";
  import StatusIcon from "./icons/StatusIcon.svelte";
  
  const dispatch = createEventDispatcher();
  
  export let isOpen = false;
  export let mode: 'modal' | 'page' = 'modal';
  
  function close() {
    if (mode === 'page') return;
    isOpen = false;
    dispatch('close');
  }
  
  $: currentTier = $userTier || 'guest';

  // Restore allPlans definition
  const allPlans = [
    {
      id: 'guest',
      name: 'Anonymous',
      price: 'Free',
      period: '1 hour',
      description: 'Instant access, no signup',
      features: [
        '1 terminal',
        '512MB memory',
        '0.5 vCPU',
        '2GB storage',
        '1 hour session limit',
        'No persistence'
      ],
      cta: 'Continue as Guest',
      current: false,
      accent: false
    },
    {
      id: 'free',
      name: 'Free',
      price: '$0',
      period: 'forever',
      description: 'For trying out Rexec with PipeOps',
      features: [
        '5 terminals',
        '2GB memory',
        '2 vCPU',
        '10GB storage',
        '50 hour session limit',
        'Community support'
      ],
      cta: 'Upgrade to Free',
      current: false,
      accent: false
    },
    {
      id: 'pro',
      name: 'Pro',
      price: '$12',
      period: '/month',
      description: 'For individual developers',
      features: [
        '10 terminals',
        '4GB memory',
        '4 vCPU',
        '20GB storage',
        'Unlimited sessions',
        'Recording & playback',
        'Share with 5 collaborators',
        'Priority support'
      ],
      cta: 'Upgrade to Pro',
      current: false,
      accent: true
    },
    {
      id: 'enterprise',
      name: 'Team',
      price: '$49',
      period: '/month',
      description: 'For teams & organizations',
      features: [
        'Unlimited terminals',
        'Up to 32GB memory',
        'Up to 16 vCPU',
        '1TB storage',
        'Persistent terminals',
        'Team recordings library',
        'Unlimited collaborators',
        'GPU access',
        'SSO & SAML',
        'Dedicated support'
      ],
      cta: 'Contact Sales',
      current: false,
      accent: false
    }
  ];

  // Helper to augment plans with current state
  $: plans = (mode === 'page' 
      ? allPlans.filter(p => p.id !== 'guest') 
      : (currentTier === 'guest' ? allPlans.filter(p => p.id !== 'free') : allPlans.filter(p => p.id !== 'guest')))
      .map(p => ({
          ...p,
          current: p.id === currentTier,
          cta: p.id === currentTier ? 'Current Plan' : p.cta
      }));
</script>

{#if isOpen || mode === 'page'}
  <div class={mode === 'modal' ? 'pricing-overlay' : 'pricing-page-wrapper'} 
       onclick={mode === 'modal' ? (e) => e.target === e.currentTarget && close() : undefined} 
       onkeydown={mode === 'modal' ? (e) => e.key === 'Escape' && close() : undefined} 
       role={mode === 'modal' ? "presentation" : undefined}
  >
    <div class={mode === 'modal' ? 'pricing-modal' : 'pricing-page-container'} role={mode === 'modal' ? "dialog" : undefined}>
      {#if mode === 'modal'}
        <button class="close-btn" onclick={close}>×</button>
      {/if}
      
      <div class="pricing-header">
        <h1>{mode === 'page' ? 'Simple, Transparent Pricing' : 'Choose Your Plan'}</h1>
        <p>Scale your terminal infrastructure as you grow</p>
      </div>
      
      <div class="plans-grid">
        {#each plans as plan}
          <div class="plan-card" class:accent={plan.accent} class:current={plan.current}>
            {#if plan.accent}
              <div class="popular-badge">Most Popular</div>
            {/if}
            
            <div class="plan-header">
              <h2>{plan.name}</h2>
              <div class="price-row">
                <span class="price">{plan.price}</span>
                <span class="period">{plan.period}</span>
              </div>
              <p class="plan-desc">{plan.description}</p>
            </div>
            
            <ul class="features">
              {#each plan.features as feature}
                <li>
                  <span class="check"><StatusIcon status="check" size={12} /></span>
                  {feature}
                </li>
              {/each}
            </ul>
            
            <button 
              class="plan-cta" 
              class:current={plan.current}
              class:accent={plan.accent}
              disabled={plan.current}
              onclick={() => {
                  if (mode === 'page') {
                      // Redirect to landing to start
                      window.location.href = '/';
                  } else {
                      // In modal, maybe just close or handle upgrade
                      close();
                  }
              }}
            >
              {plan.cta}
            </button>
          </div>
        {/each}
      </div>
      
      <div class="pricing-footer">
        <p><strong>Need more?</strong> Contact us for custom enterprise solutions.</p>
      </div>
    </div>
  </div>
{/if}

<style>
  .pricing-page-wrapper {
    width: 100%;
    min-height: 100vh;
    padding: 40px 20px;
    background: #0a0a0c;
  }

  .pricing-page-container {
    width: 100%;
    max-width: 1200px;
    margin: 0 auto;
  }

  .pricing-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.85);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: 20px;
    backdrop-filter: blur(4px);
  }
  
  .pricing-modal {
    background: #0a0a0c;
    border: 1px solid #1e1e28;
    border-radius: 8px;
    width: 100%;
    max-width: 1000px;
    max-height: 90vh;
    overflow-y: auto;
    position: relative;
    padding: 32px;
  }
  
  .close-btn {
    position: absolute;
    top: 16px;
    right: 16px;
    background: transparent;
    border: none;
    color: #999;
    font-size: 24px;
    cursor: pointer;
    padding: 4px 10px;
    border-radius: 4px;
    transition: all 0.15s;
    line-height: 1;
  }
  
  .close-btn:hover {
    color: var(--accent, #00ff88);
    background: rgba(0, 255, 136, 0.1);
  }
  
  .pricing-header {
    text-align: center;
    margin-bottom: 32px;
  }
  
  .pricing-header h1 {
    font-size: 28px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 2px;
    margin: 0 0 8px;
    color: #fff;
  }
  
  .pricing-header p {
    color: #a0a0a0;
    font-size: 14px;
    margin: 0;
  }
  
  .plans-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 20px;
    margin-bottom: 24px;
  }
  
  .plan-card {
    background: #0f0f14;
    border: 1px solid #1e1e28;
    border-radius: 6px;
    padding: 24px;
    display: flex;
    flex-direction: column;
    position: relative;
    transition: all 0.2s;
  }
  
  .plan-card:hover {
    border-color: #2a2a35;
    transform: translateY(-2px);
  }
  
  .plan-card.accent {
    border-color: var(--accent, #00ff88);
    background: rgba(0, 255, 136, 0.02);
  }
  
  .plan-card.current {
    border-color: #333;
    opacity: 0.8;
  }
  
  .popular-badge {
    position: absolute;
    top: -10px;
    left: 50%;
    transform: translateX(-50%);
    background: var(--accent, #00ff88);
    color: #000;
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    padding: 4px 12px;
    border-radius: 10px;
  }
  
  .plan-header {
    text-align: center;
    padding-bottom: 20px;
    border-bottom: 1px solid #1e1e28;
    margin-bottom: 20px;
  }
  
  .plan-header h2 {
    font-size: 18px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin: 0 0 12px;
    color: #fff;
  }
  
  .price-row {
    display: flex;
    align-items: baseline;
    justify-content: center;
    gap: 4px;
    margin-bottom: 8px;
  }
  
  .price {
    font-size: 36px;
    font-weight: 700;
    color: var(--accent, #00ff88);
    font-family: var(--font-mono, monospace);
  }
  
  .period {
    font-size: 14px;
    color: #a0a0a0;
  }
  
  .plan-desc {
    font-size: 12px;
    color: #999;
    margin: 0;
  }
  
  .features {
    list-style: none;
    padding: 0;
    margin: 0 0 20px;
    flex: 1;
  }
  
  .features li {
    display: flex;
    align-items: flex-start;
    gap: 10px;
    padding: 8px 0;
    font-size: 13px;
    color: #ccc;
    border-bottom: 1px solid #151518;
  }
  
  .features li:last-child {
    border-bottom: none;
  }
  
  .check {
    color: var(--accent, #00ff88);
    font-weight: 600;
    flex-shrink: 0;
  }
  
  .plan-cta {
    width: 100%;
    padding: 12px;
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border: 1px solid #333;
    border-radius: 4px;
    background: #1a1a1f;
    color: #999;
    cursor: pointer;
    transition: all 0.2s;
  }
  
  .plan-cta:hover:not(:disabled) {
    background: #222;
    border-color: #555;
    color: #fff;
  }
  
  .plan-cta.accent {
    background: var(--accent, #00ff88);
    border-color: var(--accent, #00ff88);
    color: #000;
  }
  
  .plan-cta.accent:hover {
    filter: brightness(1.1);
  }
  
  .plan-cta.current {
    background: transparent;
    border-color: #333;
    color: #888;
    cursor: default;
  }
  
  .pricing-footer {
    text-align: center;
    padding: 16px;
    background: rgba(0, 255, 136, 0.05);
    border: 1px solid rgba(0, 255, 136, 0.1);
    border-radius: 4px;
  }
  
  .pricing-footer p {
    margin: 0;
    font-size: 13px;
    color: #888;
  }
  
  .pricing-footer strong {
    color: var(--accent, #00ff88);
  }
  
  /* Scrollbar */
  .pricing-modal::-webkit-scrollbar {
    width: 6px;
  }
  
  .pricing-modal::-webkit-scrollbar-track {
    background: transparent;
  }
  
  .pricing-modal::-webkit-scrollbar-thumb {
    background: #222;
    border-radius: 3px;
  }
  
  .pricing-modal::-webkit-scrollbar-thumb:hover {
    background: #333;
  }
  
  /* Firefox */
  .pricing-modal {
    scrollbar-width: thin;
    scrollbar-color: #222 transparent;
  }
  
  @media (max-width: 768px) {
    .pricing-modal {
      padding: 20px;
    }
    
    .pricing-header h1 {
      font-size: 22px;
    }
    
    .plans-grid {
      grid-template-columns: 1fr;
    }
    
    .plan-card {
      padding: 20px;
    }
  }
</style>
