import { ref } from 'vue'
import { defineStore } from 'pinia'
import { onboardingApi, type OnboardingInput } from './api'
import { useSessionStore } from '../session/session.store'

export type OnboardingAvailability = 'unknown' | 'available' | 'claimed' | 'error'

export const useOnboardingStore = defineStore('onboarding', () => {
  const availability = ref<OnboardingAvailability>('unknown')
  const loaded = ref(false)

  async function loadStatus(force = false) {
    if (loaded.value && !force) return availability.value === 'available'

    try {
      const result = await onboardingApi.status()
      availability.value = result.data.available ? 'available' : 'claimed'
      loaded.value = true
      return result.data.available
    } catch (error) {
      availability.value = 'error'
      loaded.value = false
      throw error
    }
  }

  async function create(input: OnboardingInput) {
    const result = await onboardingApi.create(input)
    await useSessionStore().acceptSession(result)
    availability.value = 'claimed'
    loaded.value = true
  }

  return { availability, loaded, loadStatus, create }
})
