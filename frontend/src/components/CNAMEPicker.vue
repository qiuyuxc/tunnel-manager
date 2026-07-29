<template>
  <div ref="picker" class="cname-picker" :class="{ disabled }">
    <input
      :id="inputId"
      :value="modelValue"
      class="vercel-input cname-input"
      :placeholder="placeholder"
      :disabled="disabled"
      autocomplete="off"
      @input="handleInput"
      @keydown.arrow-down.prevent="openMenu"
      @keydown.escape="closeMenu"
    />
    <button
      ref="toggle"
      class="picker-toggle"
      type="button"
      :disabled="disabled"
      :aria-expanded="open"
      aria-label="展开优选 CNAME 选择组"
      @click="toggleMenu"
    >
      <svg :class="{ rotated: open }" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <polyline points="6 9 12 15 18 9" />
      </svg>
    </button>

    <Transition name="picker-menu">
      <div v-if="open" class="picker-menu" role="listbox">
        <button
          v-if="showDefault"
          class="picker-option default-option"
          :class="{ active: modelValue === '' }"
          type="button"
          role="option"
          :aria-selected="modelValue === ''"
          @click="selectValue('')"
        >
          <span>{{ defaultLabel }}</span>
          <small v-if="defaultValue">{{ defaultValue }}</small>
        </button>
        <button
          v-for="item in presets"
          :key="item.value"
          class="picker-option"
          :class="{ active: modelValue === item.value }"
          type="button"
          role="option"
          :aria-selected="modelValue === item.value"
          @click="selectValue(item.value)"
        >
          <span>{{ item.name }}</span>
          <small>{{ item.value }}</small>
        </button>
        <div v-if="!showDefault && presets.length === 0" class="picker-empty">暂无常用 CNAME，请前往全局设置添加。</div>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import type { CNAMEPreset } from '../api'

const props = withDefaults(defineProps<{
  modelValue: string
  presets: CNAMEPreset[]
  inputId?: string
  placeholder?: string
  disabled?: boolean
  showDefault?: boolean
  defaultLabel?: string
  defaultValue?: string
}>(), {
  inputId: undefined,
  placeholder: '选择常用线路或手动输入',
  disabled: false,
  showDefault: false,
  defaultLabel: '使用默认配置',
  defaultValue: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const picker = ref<HTMLElement | null>(null)
const toggle = ref<HTMLButtonElement | null>(null)
const open = ref(false)

function handleInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}

function openMenu() {
  if (!props.disabled) open.value = true
}

function closeMenu() {
  open.value = false
}

function toggleMenu() {
  if (props.disabled) return
  open.value = !open.value
}

function selectValue(value: string) {
  emit('update:modelValue', value)
  closeMenu()
  nextTick(() => toggle.value?.focus({ preventScroll: true }))
}

function handleOutsidePointer(event: PointerEvent) {
  if (!picker.value?.contains(event.target as Node)) closeMenu()
}

onMounted(() => document.addEventListener('pointerdown', handleOutsidePointer))
onBeforeUnmount(() => document.removeEventListener('pointerdown', handleOutsidePointer))
</script>

<style scoped>
.cname-picker { position: relative; width: 100%; }
.cname-input { padding-right: 44px; }
.picker-toggle {
  position: absolute;
  top: 1px;
  right: 1px;
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  color: var(--color-mute);
  background: transparent;
  border: 0;
  border-left: 1px solid var(--color-hairline);
  border-radius: 0 var(--radius-sm) var(--radius-sm) 0;
  cursor: pointer;
}
.picker-toggle:hover:not(:disabled) { color: var(--color-ink); background: var(--color-canvas-soft); }
.picker-toggle:disabled { cursor: not-allowed; opacity: 0.45; }
.picker-toggle svg { transition: transform 160ms ease-out; }
.picker-toggle svg.rotated { transform: rotate(180deg); }
.picker-menu {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 80;
  display: flex;
  width: 100%;
  max-height: 260px;
  flex-direction: column;
  gap: 2px;
  padding: 6px;
  overflow-y: auto;
  box-sizing: border-box;
  background: var(--color-canvas);
  border: 1px solid var(--color-hairline-strong);
  border-radius: var(--radius-md);
  box-shadow: 0 18px 42px rgba(38, 31, 22, 0.16);
}
.picker-option {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-sm);
  padding: 9px 10px;
  color: var(--color-ink);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.picker-option:hover, .picker-option.active { background: var(--color-canvas-soft); }
.picker-option.active { box-shadow: inset 2px 0 0 var(--color-link); }
.picker-option span { min-width: 0; font-size: 13px; font-weight: 700; overflow-wrap: anywhere; }
.picker-option small { min-width: 0; color: var(--color-mute); font: 11px/1.4 var(--font-mono); overflow-wrap: anywhere; text-align: right; }
.default-option { border-bottom: 1px solid var(--color-hairline); border-radius: var(--radius-sm) var(--radius-sm) 0 0; }
.picker-empty { padding: var(--spacing-md); color: var(--color-mute); font-size: 13px; text-align: center; }
.picker-menu-enter-active, .picker-menu-leave-active { transition: opacity 140ms ease-out, transform 140ms ease-out; }
.picker-menu-enter-from, .picker-menu-leave-to { opacity: 0; transform: translateY(-4px); }
</style>
