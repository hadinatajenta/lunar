<template>
  <div class="detail-section">
    <p class="detail-section-label">AI output</p>
    <div class="ai-output">
      <div class="ai-output-head">
        <span class="ai-badge">AI</span>
        Lunar Copilot
      </div>
      <div class="ai-empty">
        Click “Generate with AI” to analyze the source and produce documentation. The result
        will appear here.
      </div>
    </div>
  </div>
</template>

<style scoped>
.detail-section {
  margin-bottom: 24px;
}

.detail-section-label {
  margin: 0 0 10px;
  color: var(--subtle);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.ai-output {
  padding: 16px 18px;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.018);
}

.ai-output-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  color: var(--muted);
  font-size: 10px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.ai-badge {
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  flex: 0 0 auto;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.07);
  color: #cfd6de;
  font-size: 9px;
  font-weight: 700;
}

.ai-empty {
  color: var(--subtle);
  font-size: 12px;
  line-height: 1.6;
}
</style>
