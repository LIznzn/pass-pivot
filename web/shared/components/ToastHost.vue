<template>
  <div class="ppvt-toast-stack" aria-live="polite" aria-atomic="true">
    <transition-group name="ppvt-toast">
      <div
        v-for="item in toast.items"
        :key="item.id"
        class="ppvt-toast"
        :class="`is-${item.variant}`"
        role="status"
      >
        <span class="ppvt-toast-icon" aria-hidden="true">{{ iconMap[item.variant] }}</span>
        <span class="ppvt-toast-message">{{ item.message }}</span>
      </div>
    </transition-group>
  </div>
</template>

<script setup lang="ts">
import { useToast } from '../composables/toast'

const toast = useToast()

const iconMap = {
  success: '✓',
  danger: '✕',
  warning: '!',
  info: 'i'
} as const
</script>
