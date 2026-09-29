<template>
  <section class="hero" aria-label="Lunar introduction">
    <div class="hero-content">
      <div class="brand-row">
        <router-link to="/" class="brand" aria-label="Lunar home">
          <span class="brand-mark" aria-hidden="true" />
          <span>Lunar</span>
        </router-link>
      </div>

      <p class="eyebrow">Developer workspace</p>

      <h1>Work closer to the signal.</h1>

      <p class="hero-copy">
        Connect your engineering tools, explore context, and work with AI
        from one focused workspace.
      </p>

      <div class="signal">
        <span class="signal-dot" aria-hidden="true" />
        <span>All systems operational</span>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
</script>

<style scoped>
.hero {
  min-width: 0;
  display: flex;
  align-items: center;
  padding: 64px clamp(40px, 8vw, 128px);
}

.hero-content {
  width: min(100%, 680px);
}

.brand-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 72px;
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--text);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: -0.02em;
  text-decoration: none;
}

.brand-mark {
  position: relative;
  width: 26px;
  height: 26px;
  display: grid;
  place-items: center;
  border: 1px solid rgba(255, 255, 255, 0.25);
  border-radius: 50%;
}

.brand-mark::before {
  content: "";
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 18px rgba(245, 247, 250, 0.24);
}

.eyebrow {
  margin: 0 0 18px;
  color: var(--muted);
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

h1 {
  max-width: 620px;
  margin: 0;
  font-size: clamp(52px, 6.4vw, 88px);
  font-weight: 600;
  line-height: 0.96;
  letter-spacing: -0.06em;
}

.hero-copy {
  max-width: 500px;
  margin: 28px 0 0;
  color: var(--muted);
  font-size: 16px;
  line-height: 1.6;
}

.signal {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 34px;
  color: var(--subtle);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.signal-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #bcc4ce;
  box-shadow: 0 0 12px rgba(220, 226, 235, 0.34);
}

@media (max-width: 980px) {
  .hero {
    min-height: 390px;
    padding: 48px 28px;
  }

  .brand-row {
    margin-bottom: 56px;
  }

  .hero-copy,
  .signal {
    display: none;
  }
}

@media (max-width: 520px) {
  .hero {
    min-height: 300px;
    padding: 32px 22px;
  }

  .brand-row {
    margin-bottom: 44px;
  }

  h1 {
    font-size: clamp(46px, 15vw, 66px);
  }
}
</style>
