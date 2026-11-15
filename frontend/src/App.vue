<template>
  <router-view v-if="route.name === 'public-status'" />
  <ConsoleApp v-else />
</template>

<script setup lang="ts">
import { defineAsyncComponent } from 'vue'
import { useRoute } from 'vue-router'

// Keep console providers, navigation and account bootstrap out of public status.
const ConsoleApp = defineAsyncComponent(() => import('./ConsoleApp.vue'))
const route = useRoute()
</script>
