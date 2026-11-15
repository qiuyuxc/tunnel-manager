<template>
  <button type="button" class="refresh-button" :disabled="busy || disabled"
    :aria-label="busy ? tr('m_3af7e5489e61') : tr('m_aee887434131')"
    :aria-busy="busy" @click="$emit('refresh')">
    <svg :class="{ 'is-busy': busy }" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d="M20 7v5h-5M4 17v-5h5" />
      <path d="M5.7 7a7.5 7.5 0 0 1 12.4-1.2L20 8M4 16l1.9 2.2A7.5 7.5 0 0 0 18.3 17" />
    </svg>
    <span v-if="!compact">{{ busy ? tr('m_3af7e5489e61') : tr('m_aee887434131') }}</span>
  </button>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'
defineProps<{ busy?: boolean; disabled?: boolean; compact?: boolean }>()
defineEmits<{ refresh: [] }>()
</script>
<style scoped>
.refresh-button { display: inline-flex; justify-content: center; align-items: center; flex-shrink: 0; gap: 8px; min-width: 42px; min-height: 42px; box-sizing: border-box; padding: 8px 13px; color: var(--color-body); background: var(--color-canvas); border: 1px solid var(--color-hairline); border-radius: 12px; font: inherit; font-size: 12px; cursor: pointer; }
.refresh-button:hover:not(:disabled) { color: var(--color-ink); background: var(--color-canvas-soft-2); border-color: var(--color-hairline-strong); }
.refresh-button:disabled { cursor: default; opacity: .6; }
.is-busy { animation: refresh-spin 1s linear infinite; }
@keyframes refresh-spin { to { transform: rotate(360deg); } }
@media (prefers-reduced-motion: reduce) { .is-busy { animation: none; } }
</style>
