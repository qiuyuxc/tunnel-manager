<template>
  <Teleport to="body">
    <Transition name="sheet-scrim">
      <div v-if="open" class="sheet-scrim" @click="emit('close')" />
    </Transition>
    <Transition name="sheet-rise">
      <div v-if="open" class="sheet" role="dialog" aria-modal="true" :aria-label="title">
        <span class="sheet-handle" aria-hidden="true" />
        <div class="sheet-head">
          <span class="sheet-title">{{ title }}</span>
          <button type="button" class="sheet-close" aria-label="关闭" @click="emit('close')">
            <span v-html="icons.close" />
          </button>
        </div>
        <div class="sheet-body"><slot /></div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, watch } from 'vue'
import { icons } from '../navigation'

const props = defineProps<{ open: boolean; title: string }>()
const emit = defineEmits<{ close: [] }>()

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') emit('close')
}

watch(
  () => props.open,
  (open) => {
    document.body.style.overflow = open ? 'hidden' : ''
    if (open) window.addEventListener('keydown', onKeydown)
    else window.removeEventListener('keydown', onKeydown)
  },
)

onBeforeUnmount(() => {
  document.body.style.overflow = ''
  window.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.sheet-scrim {
  position: fixed;
  inset: 0;
  z-index: 199;
  background: rgba(0, 0, 0, 0.45);
}

.sheet {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 200;
  max-height: 88vh;
  display: flex;
  flex-direction: column;
  background: var(--color-canvas-raised);
  border-top: 1px solid var(--color-hairline);
  border-radius: 18px 18px 0 0;
  box-shadow: 0 -8px 32px rgba(0, 0, 0, 0.2);
}

.sheet-handle {
  flex: none;
  width: 36px;
  height: 4px;
  margin: 9px auto 0;
  border-radius: 2px;
  background: var(--color-hairline-strong);
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
  width: 28px;
  height: 28px;
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
    max-height: 80vh;
    border: 1px solid var(--color-hairline);
    border-radius: 18px;
    box-shadow: 0 24px 64px rgb(0 0 0 / 30%);
  }

  .sheet-handle { display: none; }
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
