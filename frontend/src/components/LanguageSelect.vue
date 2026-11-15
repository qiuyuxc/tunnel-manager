<template>
  <div class="language-select">
    <span>{{ tr('language.label') }}</span>
    <n-select
      class="language-control"
      :value="languagePreference"
      :options="options"
      :aria-label="tr('language.label')"
      size="small"
      @update:value="selectPreference"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NSelect } from 'naive-ui'
import { languagePreference, setLanguage, t as tr, type LanguagePreference } from '../i18n'

const options = computed(() => [
  { value: 'system', label: tr('language.system') },
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en', label: 'English' },
])
function selectPreference(value: unknown) {
  if (value === 'system' || value === 'zh-CN' || value === 'en') setLanguage(value as LanguagePreference)
}
</script>

<style scoped>
.language-select { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 12px; color: var(--color-ink); font-size: 13px; }
.language-select > span { flex-shrink: 0; }
.language-control { flex: 1; min-width: 116px; max-width: 160px; }
.language-control :deep(.n-base-selection) { font-size: 12px; }
</style>
