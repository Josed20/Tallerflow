import { fireEvent, render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import MetricTrend from './MetricTrend.vue'

describe('MetricTrend', () => {
  it('keeps missing days separate from zero observations and never connects across a gap', async () => {
    const { container } = render(MetricTrend, { props: { rows: [
      { day: '2026-11-23', active_users: 0, first_opens: 0, registrations: 0, play_device_installs: null },
      { day: '2026-11-24', active_users: null, first_opens: null, registrations: null, play_device_installs: null },
      { day: '2026-11-25', active_users: 3, first_opens: 1, registrations: 1, play_device_installs: null },
    ] } })
    expect(screen.getByRole('img').getAttribute('aria-label')).toContain('Máximo diario: 3')
    expect(container.querySelectorAll('.trend-line')).toHaveLength(2)
    expect(container.querySelectorAll('circle')).toHaveLength(2)
    await fireEvent.update(screen.getByLabelText('Métrica del gráfico'), 'play_device_installs')
    expect(screen.queryByRole('img')).toBeNull()
    expect(screen.getByText(/Todavía no hay datos/)).toBeTruthy()
  })

  it('reports an observed maximum of zero without inventing activity', () => {
    render(MetricTrend, { props: { rows: [{ day: '2026-11-25', active_users: 0, first_opens: 0, registrations: 0, play_device_installs: 0 }] } })
    expect(screen.getByRole('img').getAttribute('aria-label')).toContain('Máximo diario: 0')
    expect(screen.queryByText(/Todavía no hay datos/)).toBeNull()
  })
})
