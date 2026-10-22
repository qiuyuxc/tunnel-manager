<template>
  <details v-if="active || reasoning" class="thinking-details">
    <summary><span v-if="active" class="pulse" aria-hidden="true"></span>{{ active ? (replying ? tr('m_85fd04d0c89c') : tr('m_dfd317d5828c')) : tr('m_c71d5942ebbc') }}<span class="chevron" aria-hidden="true">⌄</span></summary>
    <p v-if="reasoning">{{ reasoning }}</p>
    <p v-else>{{ status || tr('m_ea67d9f548d5') }}</p>
  </details>
</template>
<script setup lang="ts">
import { t as tr } from '../i18n'

defineProps<{ active?: boolean; reasoning?: string; status?: string; replying?: boolean }>()
</script>
<style scoped>
.thinking-details { font-size: 12px; color: var(--color-mute); margin: 8px 0; }
summary { display: flex; align-items: center; gap: 8px; min-height: 36px; cursor: pointer; list-style: none; width: fit-content; }
summary::-webkit-details-marker { display: none; }
.chevron { display: inline-block; transition: transform .15s; }
[open] .chevron { transform: rotate(180deg); }
p { margin: 4px 0 12px; padding-left: 12px; border-left: 2px solid var(--color-hairline); white-space: pre-wrap; overflow-wrap: anywhere; max-height: 220px; overflow-y: auto; line-height: 1.7; }
.pulse { width: 6px; height: 6px; border-radius: 50%; background: var(--color-success); animation: thinking-pulse 1.5s ease-in-out infinite; }
@keyframes thinking-pulse { 50% { opacity: .25; } }
@media (prefers-reduced-motion: reduce) { .pulse { animation: none; } }
</style>
