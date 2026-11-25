<script setup lang="ts">
import FeedbackToast from './FeedbackToast.vue'
defineProps<{ message: string }>()
</script>

<template>
  <FeedbackToast kind="error" :message="message" />
</template>
