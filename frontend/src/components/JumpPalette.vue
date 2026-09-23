<template>
  <!-- Quick-jump palette (⌘K / Ctrl+K). Lists the same permission-filtered nav
       destinations the sidebar shows, so it can only ever navigate where the
       account is already allowed to go — it never invents a route. -->
  <Transition name="jump-fade">
    <div v-if="open" class="jump-scrim" @click="close">
      <div class="jump-panel" role="dialog" aria-modal="true" aria-label="快速前往" @click.stop>
        <div class="jump-search">
          <span class="nav-icon" aria-hidden="true" v-html="icons.search" />
          <input
            ref="inputEl"
            v-model="query"
            type="text"
            placeholder="搜索页面…"
            aria-label="搜索页面"
            @keydown.down.prevent="move(1)"
            @keydown.up.prevent="move(-1)"
            @keydown.enter.prevent="choose(results[active])"
            @keydown.esc.prevent="close"
          />
          <kbd>esc</kbd>
        </div>
        <div class="jump-results">
          <button
            v-for="(item, i) in results"
            :key="item.path"
            type="button"
            class="jump-row"
            :class="{ active: i === active }"
            @mousemove="active = i"
            @click="choose(item)"
          >
            <span class="jump-icon" aria-hidden="true" v-html="item.icon" />
            <span class="jump-label">{{ item.label }}</span>
            <span class="jump-group">{{ groupLabel(item.group) }}</span>
          </button>
          <p v-if="!results.length" class="jump-empty">没有匹配的页面</p>
        </div>
      </div>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { GROUP_LABELS, icons, useNavItems, type NavGroup, type NavItem } from '../navigation'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()

const router = useRouter()
const navItems = useNavItems()
const query = ref('')
const active = ref(0)
const inputEl = ref<HTMLInputElement | null>(null)

const results = computed(() => {
  const q = query.value.trim().toLowerCase()
  const list = navItems.value
  if (!q) return list
  return list.filter((item) => item.label.toLowerCase().includes(q) || item.path.toLowerCase().includes(q))
})

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    query.value = ''
    active.value = 0
    nextTick(() => inputEl.value?.focus())
  }
})

watch(results, () => { active.value = 0 })

function groupLabel(group: NavGroup) {
  return GROUP_LABELS[group]
}

function move(delta: number) {
  const count = results.value.length
  if (!count) return
  active.value = (active.value + delta + count) % count
}

function choose(item: NavItem | undefined) {
  if (!item) return
  emit('close')
  router.push(item.path)
}

function close() {
  emit('close')
}
</script>

<style scoped>
.jump-scrim {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 12vh 16px 16px;
  background: rgb(0 8 5 / 55%);
}

.jump-panel {
  width: min(560px, 100%);
  overflow: hidden;
  background: var(--color-canvas-raised);
  border: 1px solid var(--color-hairline);
  border-radius: 18px;
  box-shadow: 0 24px 64px rgb(0 0 0 / 30%);
}

.jump-search {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-hairline);
}

.jump-search .nav-icon { display: inline-flex; color: var(--color-mute); }
.jump-search input {
  flex: 1;
  min-width: 0;
  border: 0;
  background: transparent;
  color: var(--color-ink);
  font: inherit;
  font-size: 15px;
  outline: none;
}
.jump-search kbd {
  padding: 1px 6px;
  border: 1px solid var(--color-hairline);
  border-radius: 5px;
  font: 11px inherit;
  color: var(--color-mute);
}

.jump-results { max-height: 46vh; overflow-y: auto; padding: 8px; }

.jump-row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px 12px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}
.jump-row.active { background: var(--color-sidebar-active-bg); color: var(--color-sidebar-text-active); }
.jump-icon { display: inline-flex; flex: 0 0 auto; color: var(--color-body); }
.jump-row.active .jump-icon { color: var(--color-sidebar-text-active); }
.jump-icon :deep(svg) { width: 18px; height: 18px; }
.jump-label { flex: 1; min-width: 0; }
.jump-group { font-size: 11px; color: var(--color-mute); }
.jump-empty { padding: 24px; text-align: center; color: var(--color-mute); font-size: 13px; }

.jump-fade-enter-active, .jump-fade-leave-active { transition: opacity 150ms ease; }
.jump-fade-enter-from, .jump-fade-leave-to { opacity: 0; }

@media (prefers-reduced-motion: reduce) {
  .jump-fade-enter-active, .jump-fade-leave-active { transition: none; }
}
</style>
