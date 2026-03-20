<template>
  <div class="admin-page-header">
    <div>
      <h2 v-if="title" class="section-page-title">{{ title }}</h2>
      <p v-if="description" class="section-page-subtitle">{{ description }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description: string
}>()
</script>
