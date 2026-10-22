<template>
  <label class="language-select">
    <span>{{ tr('language.label') }}</span>
    <select :value="languagePreference" @change="setLanguage(($event.target as HTMLSelectElement).value as LanguagePreference)">
      <option value="system">{{ tr('language.system') }}</option>
      <option value="zh-CN">简体中文</option>
      <option value="en">English</option>
    </select>
  </label>
</template>

<script setup lang="ts">
import { languagePreference, setLanguage, t as tr, type LanguagePreference } from '../i18n'
</script>

<style scoped>
.language-select { display:flex;align-items:center;justify-content:space-between;gap:12px;padding:9px 12px;color:var(--color-ink);font-size:14px }
select { min-width:0;max-width:160px;min-height:36px;border:1px solid var(--color-hairline);border-radius:8px;padding:6px;background:var(--color-canvas-raised);color:inherit;font:inherit }
</style>
