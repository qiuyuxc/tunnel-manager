<template>
  <section class="operation-status" role="status" aria-live="polite" aria-busy="true">
    <span class="operation-spinner spin" aria-hidden="true"></span>
    <div><strong>{{ title }}</strong><p>{{ elapsed >= 15 ? '仍未收到服务端结果，请继续等待，不要重复提交。' : '请求已发出，正在等待服务端返回结果。' }}</p><span aria-hidden="true">已等待 {{ elapsed }} 秒</span></div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
defineProps<{ title: string }>()
const startedAt = Date.now()
const elapsed = ref(0)
const timer = setInterval(() => { elapsed.value = Math.floor((Date.now() - startedAt) / 1000) }, 1000)
onBeforeUnmount(() => clearInterval(timer))
</script>

<style scoped>
.operation-status{display:flex;align-items:flex-start;gap:12px;padding:16px;margin:16px 0;background:var(--color-canvas-soft);border:1px solid var(--color-hairline);border-radius:var(--radius-md)}.operation-status>div{min-width:0}.operation-status strong{font-size:14px;color:var(--color-ink)}.operation-status p{font-size:12px;color:var(--color-body);margin:6px 0;line-height:1.6}.operation-status span{font-size:12px;color:var(--color-mute);font-variant-numeric:tabular-nums}.operation-spinner{flex:none;width:18px;height:18px;margin-top:1px;border:2px solid var(--color-hairline);border-top-color:var(--color-success);border-radius:50%}
@media (prefers-reduced-motion: reduce) { .operation-spinner { animation: none !important; } }
</style>
