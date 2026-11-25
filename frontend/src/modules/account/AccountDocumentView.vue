<script setup lang="ts">
import { ArrowLeft } from '@lucide/vue'
import TermsView from '../legal/TermsView.vue'
import PrivacyView from '../legal/PrivacyView.vue'
import DeleteAccountView from '../legal/DeleteAccountView.vue'
defineProps<{ kind: 'privacy' | 'deletion' | 'terms' }>()
</script>
<template>
  <main class="account-document-page">
    <RouterLink class="back-link" to="/app/account"><ArrowLeft :size="18" /> Mi cuenta</RouterLink>
    <div class="document-surface"><PrivacyView v-if="kind === 'privacy'" embedded /><DeleteAccountView v-else-if="kind === 'deletion'" embedded /><TermsView v-else embedded /></div>
  </main>
</template>
<style scoped>
.account-document-page{max-width:960px;margin:auto;padding:clamp(1rem,3vw,2.5rem)}
.back-link{display:inline-flex;align-items:center;gap:.5rem;min-height:44px;margin-bottom:1rem;text-decoration:none;color:var(--tf-primary);font-weight:600}
.document-surface{padding:clamp(1.2rem,4vw,2.5rem);border:1px solid var(--tf-border-soft);border-radius:18px;background:white}
.document-surface :deep(.legal-page){min-height:0;background:transparent}.document-surface :deep(.legal-container){width:100%;max-width:none;padding:0}.document-surface :deep(.legal-article){border:0;padding:0;background:transparent;box-shadow:none}.document-surface :deep(h1){font-size:clamp(1.6rem,4vw,2rem)}.document-surface :deep(h2){font-size:1.15rem;margin-top:1.75rem}.document-surface :deep(.legal-eyebrow){display:none}
</style>
