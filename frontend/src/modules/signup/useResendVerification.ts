import { onBeforeUnmount, ref } from 'vue'
import { ApiError } from '../session/api'
import { signupApi } from './api'

// The server answers the same for any address, so the screen only says that
// a link is on its way. The cooldown keeps an impatient double tap from
// spending the per-IP budget.
export function useResendVerification(cooldownSeconds = 60) {
  const sending = ref(false)
  const sent = ref(false)
  const failed = ref('')
  const remaining = ref(0)
  let timer: ReturnType<typeof setInterval> | undefined

  function stop() {
    if (timer !== undefined) clearInterval(timer)
    timer = undefined
    remaining.value = 0
  }

  function startCooldown() {
    stop()
    remaining.value = cooldownSeconds
    timer = setInterval(() => {
      remaining.value -= 1
      if (remaining.value <= 0) stop()
    }, 1000)
  }

  async function resend(email: string) {
    if (sending.value || remaining.value > 0) return
    sending.value = true
    failed.value = ''
    try {
      await signupApi.resend(email.trim())
      sent.value = true
      startCooldown()
    } catch (error) {
      failed.value = error instanceof ApiError && error.code === 'SIGNUP_RATE_LIMITED'
        ? 'Pediste muchos enlaces seguidos. Espera unos minutos e inténtalo de nuevo.'
        : 'No pudimos reenviar el correo. Inténtalo de nuevo en un momento.'
    } finally {
      sending.value = false
    }
  }

  onBeforeUnmount(stop)

  return { sending, sent, failed, remaining, resend }
}
