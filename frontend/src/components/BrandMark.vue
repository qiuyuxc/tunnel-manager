<template>
  <!-- The product mark: the site icon when the operator uploaded one, otherwise
       the tunnel-arch glyph on a mint rounded square (same glyph as the favicon
       in index.html, so the browser tab and the sidebar agree). Sized through a
       CSS var so callers only pass one number. -->
  <span class="brand-mark" :style="{ '--mark-size': size + 'px' }" aria-hidden="true">
    <img v-if="configStore.config.site_icon" :src="configStore.config.site_icon" alt="" />
    <svg v-else viewBox="0 0 32 32" fill="none">
      <path
        d="M9 22V13a7 7 0 0 1 14 0v9M14 22v-8a2 2 0 0 1 4 0v8"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  </span>
</template>

<script setup lang="ts">
import { useConfigStore } from '../stores/config'

withDefaults(defineProps<{ size?: number }>(), { size: 36 })
const configStore = useConfigStore()
</script>

<style scoped>
.brand-mark {
  display: grid;
  place-items: center;
  width: var(--mark-size);
  height: var(--mark-size);
  flex-shrink: 0;
  border-radius: calc(var(--mark-size) / 3);
  background: var(--color-btn-primary-bg);
  color: var(--color-btn-primary-text);
  overflow: hidden;
}

.brand-mark svg {
  width: calc(var(--mark-size) * 0.64);
  height: calc(var(--mark-size) * 0.64);
}

.brand-mark img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
