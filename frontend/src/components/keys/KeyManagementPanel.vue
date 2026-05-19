<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div class="key-panel">
    <div class="panel-header">
      <span class="panel-title">{{ t('keys.title') }}</span>
      <button class="panel-action-btn" disabled>+</button>
    </div>
    <div class="panel-body-empty">
      <span class="empty-text">{{ t('keys.comingSoon') }}</span>
    </div>
  </div>
</template>

<style scoped>
.key-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--bg-secondary);
}

.panel-header {
  padding: 10px 12px;
  background: var(--bg-tertiary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
}

.panel-title {
  font-size: var(--font-size-base);
  font-weight: 600;
  color: var(--text-primary);
}

.panel-action-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  cursor: pointer;
  padding: 2px 8px;
  border-radius: 3px;
}

.panel-action-btn:hover:not(:disabled) {
  color: var(--text-primary);
  background: var(--hover-overlay);
}

.panel-body-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.empty-text {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}
</style>
