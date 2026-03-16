<template>
  <div class="shell">
    <ToastHost />
    <div class="shell-panel">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import ToastHost from './ToastHost.vue'
</script>
