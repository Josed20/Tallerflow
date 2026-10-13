import { render, screen } from '@testing-library/vue'
import { describe, expect, it } from 'vitest'
import BrandLogo from './BrandLogo.vue'

describe('BrandLogo', () => {
  it('uses the horizontal logo for light backgrounds by default', () => {
    render(BrandLogo)
    expect(screen.getByRole('img', { name: 'TallerFlow' }).getAttribute('src')).toBe('/brand/logo.svg')
  })

  it('uses the inverse symbol on dark backgrounds when compact', () => {
    render(BrandLogo, { props: { inverse: true, symbolOnly: true } })
    const logo = screen.getByRole('img', { name: 'TallerFlow' })
    expect(logo.getAttribute('src')).toBe('/brand/symbol-on-dark.svg')
    expect(logo.getAttribute('width')).toBe('64')
  })
})
