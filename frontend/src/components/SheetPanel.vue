<template>
  <Teleport to="body">
    <Transition name="sheet-scrim">
      <div v-if="open" class="sheet-scrim" :style="sheetStyle" @click="emit('close')" />
    </Transition>
    <Transition name="sheet-rise">
      <div v-if="open" ref="sheet" class="sheet" :data-size="resolvedSize" :style="sheetStyle" tabindex="-1" role="dialog" aria-modal="true" :aria-label="title">
        <div class="sheet-head">
          <span class="sheet-title">{{ title }}</span>
          <button type="button" class="sheet-close" :aria-label="tr('m_3fd47edce45b')" @click="emit('close')">
            <span v-html="icons.close" />
          </button>
        </div>
        <div class="sheet-body"><div class="sheet-content"><slot /></div></div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { t as tr } from '../i18n'

import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { icons } from '../navigation'
import { acquireSheet, isTopSheet, releaseSheet, sheetLayer } from '../utils/sheetStack'

type SheetSize = 'compact' | 'medium' | 'large'
const props = defineProps<{ open: boolean; title: string; size: SheetSize }>()
const emit = defineEmits<{ close: [] }>()
const sheet = ref<HTMLElement | null>(null)
const viewportHeight = ref(typeof window === 'undefined' ? 800 : window.innerHeight || 800)
const visibleHeight = ref(viewportHeight.value)
const keyboardBottom = ref(0)
const ratios = { compact: .3, medium: .6, large: .8 }
const resolvedSize = computed(() => props.size)
const sheetStyle = computed(() => ({
  '--sheet-z': 200 + sheetLayer(stackToken) * 2,
  '--sheet-height': `${Math.round(viewportHeight.value * ratios[resolvedSize.value])}px`,
  '--sheet-available': `${Math.max(0, visibleHeight.value - 8)}px`,
  '--sheet-bottom': `${keyboardBottom.value}px`,
}))
const stackToken = Symbol('sheet')
let returnFocus: HTMLElement | null = null
let active = false

function onKeydown(event: KeyboardEvent) {
  if (!props.open || !isTopSheet(stackToken)) return
  if (event.key === 'Escape') { event.preventDefault(); emit('close') }
  if (event.key !== 'Tab' || !sheet.value) return
  const controls = Array.from(sheet.value.querySelectorAll<HTMLElement>('button:not(:disabled), [href], input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [tabindex]:not([tabindex="-1"])')).filter(element => element.getClientRects().length > 0)
  const first = controls[0], last = controls[controls.length - 1]
  if (!first) { event.preventDefault(); sheet.value.focus(); return }
  if (!sheet.value.contains(document.activeElement) || document.activeElement === sheet.value) {
    event.preventDefault(); (event.shiftKey ? last : first)?.focus()
  } else if (event.shiftKey && document.activeElement === first) {
    event.preventDefault(); last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault(); first.focus()
  }
}

function release() {
  if (!active) return
  active = false
  const wasTop = isTopSheet(stackToken)
  releaseSheet(stackToken)
  window.removeEventListener('keydown', onKeydown)
  if (wasTop && returnFocus?.isConnected) returnFocus.focus()
  returnFocus = null
}

watch(
  () => props.open,
  async (open) => {
    if (!open) { release(); return }
    returnFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    active = true
    acquireSheet(stackToken)
    window.addEventListener('keydown', onKeydown)
    await nextTick()
    if (props.open) sheet.value?.querySelector<HTMLElement>('input:not(:disabled), button:not(:disabled)')?.focus()
  },
  { immediate: true },
)

function measure() {
  if (!props.open) return
  viewportHeight.value = document.documentElement.clientHeight || window.innerHeight
  const viewport = window.visualViewport
  visibleHeight.value = viewport?.height || viewportHeight.value
  keyboardBottom.value = viewport ? Math.max(0, viewportHeight.value - viewport.height - viewport.offsetTop) : 0
  const focused = document.activeElement
  if (keyboardBottom.value > 0 && focused instanceof HTMLElement && sheet.value?.contains(focused)) {
    void nextTick(() => { if (props.open) focused.scrollIntoView({ block: 'nearest', inline: 'nearest' }) })
  }
}

function stopSizing() {
  window.removeEventListener('resize', measure)
  window.visualViewport?.removeEventListener('resize', measure)
  window.visualViewport?.removeEventListener('scroll', measure)
}

watch(() => [props.open, props.size], async () => {
  if (typeof window === 'undefined') return
  stopSizing()
  if (!props.open) return
  await nextTick()
  if (!props.open) return
  measure()
  window.addEventListener('resize', measure)
  window.visualViewport?.addEventListener('resize', measure)
  window.visualViewport?.addEventListener('scroll', measure)
}, { immediate: true })

onBeforeUnmount(() => { if (typeof window !== 'undefined') stopSizing(); release() })
</script>

<style scoped>
.sheet-scrim {
  position: fixed;
  inset: 0;
  z-index: calc(var(--sheet-z, 200) - 1);
  background: rgba(0, 0, 0, 0.45);
}

.sheet {
  position: fixed;
  left: 0;
  right: 0;
  bottom: var(--sheet-bottom, 0px);
  z-index: var(--sheet-z, 200);
  height: var(--sheet-height, 60dvh);
  max-height: var(--sheet-available, calc(100dvh - 8px));
  display: flex;
  flex-direction: column;
  background: var(--color-canvas-raised);
  border-top: 1px solid var(--color-hairline);
  border-radius: 18px 18px 0 0;
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.2);
}


.sheet-head {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px 10px;
}

.sheet-title { font-size: 15px; font-weight: 600; color: var(--color-ink); }

.sheet-close {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: var(--color-canvas-soft-2);
  color: var(--color-mute);
  cursor: pointer;
}

.sheet-close :deep(svg) { width: 14px; height: 14px; }

.sheet-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 0 12px calc(20px + env(safe-area-inset-bottom));
  overscroll-behavior: contain;
}
.sheet-content { display: flow-root; }

.sheet-scrim-enter-active,
.sheet-scrim-leave-active { transition: opacity 180ms ease; }
.sheet-scrim-enter-from,
.sheet-scrim-leave-to { opacity: 0; }

.sheet-rise-enter-active,
.sheet-rise-leave-active { transition: transform 220ms cubic-bezier(0.32, 0.72, 0, 1); }
.sheet-rise-enter-from,
.sheet-rise-leave-to { transform: translateY(100%); }

/* Desktop: the bottom sheet reads as mobile chrome on a wide screen, so present
   it as a centred dialog instead — same content, same close/esc behaviour. */
@media (min-width: 769px) {
  .sheet {
    left: 50%;
    right: auto;
    top: 50%;
    bottom: auto;
    transform: translate(-50%, -50%);
    width: min(480px, calc(100vw - 32px));
    height: auto;
    max-height: min(var(--sheet-height, 80dvh), var(--sheet-available, 80dvh));
    border: 1px solid var(--color-hairline);
    border-radius: 18px;
    box-shadow: 0 24px 64px rgb(0 0 0 / 30%);
  }

  .sheet-head { padding: 18px 20px 12px; }
  .sheet-body { padding: 0 20px 20px; }

  .sheet-rise-enter-active,
  .sheet-rise-leave-active { transition: transform 180ms ease, opacity 180ms ease; }
  .sheet-rise-enter-from,
  .sheet-rise-leave-to { transform: translate(-50%, -50%) scale(0.96); opacity: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .sheet-rise-enter-active,
  .sheet-rise-leave-active,
  .sheet-scrim-enter-active,
  .sheet-scrim-leave-active { transition: none; }
}
</style>
