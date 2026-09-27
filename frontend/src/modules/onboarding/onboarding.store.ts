import { ref } from 'vue'
import { defineStore } from 'pinia'
import { onboardingApi, type OnboardingInput } from './api'
import { ApiError } from '../session/api'
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
    let result
    try {
      result = await onboardingApi.create(input)
    } catch (error) {
      if (error instanceof ApiError && (error.status === 409 || error.code === 'ONBOARDING_UNAVAILABLE')) {
        availability.value = 'claimed'
        loaded.value = true
      } else {
        try {
          await loadStatus(true)
        } catch {
          // Preserve the original creation error when availability cannot be refreshed.
        }
      }
      throw error
    }
    availability.value = 'claimed'
    loaded.value = true
    await useSessionStore().acceptSession(result)
  }

  return { availability, loaded, loadStatus, create }
})
