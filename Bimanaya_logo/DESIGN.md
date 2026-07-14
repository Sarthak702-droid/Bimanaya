---
name: Clinical Integrity
colors:
  surface: '#fcf8fa'
  surface-dim: '#dcd9db'
  surface-bright: '#fcf8fa'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#f6f3f5'
  surface-container: '#f0edef'
  surface-container-high: '#eae7e9'
  surface-container-highest: '#e4e2e4'
  on-surface: '#1b1b1d'
  on-surface-variant: '#45464d'
  inverse-surface: '#303032'
  inverse-on-surface: '#f3f0f2'
  outline: '#76777d'
  outline-variant: '#c6c6cd'
  surface-tint: '#565e74'
  primary: '#000000'
  on-primary: '#ffffff'
  primary-container: '#131b2e'
  on-primary-container: '#7c839b'
  inverse-primary: '#bec6e0'
  secondary: '#006a61'
  on-secondary: '#ffffff'
  secondary-container: '#86f2e4'
  on-secondary-container: '#006f66'
  tertiary: '#000000'
  on-tertiary: '#ffffff'
  tertiary-container: '#2a1700'
  on-tertiary-container: '#b87500'
  error: '#ba1a1a'
  on-error: '#ffffff'
  error-container: '#ffdad6'
  on-error-container: '#93000a'
  primary-fixed: '#dae2fd'
  primary-fixed-dim: '#bec6e0'
  on-primary-fixed: '#131b2e'
  on-primary-fixed-variant: '#3f465c'
  secondary-fixed: '#89f5e7'
  secondary-fixed-dim: '#6bd8cb'
  on-secondary-fixed: '#00201d'
  on-secondary-fixed-variant: '#005049'
  tertiary-fixed: '#ffddb8'
  tertiary-fixed-dim: '#ffb95f'
  on-tertiary-fixed: '#2a1700'
  on-tertiary-fixed-variant: '#653e00'
  background: '#fcf8fa'
  on-background: '#1b1b1d'
  surface-variant: '#e4e2e4'
typography:
  display-lg:
    fontFamily: Geist
    fontSize: 48px
    fontWeight: '700'
    lineHeight: 56px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Geist
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.01em
  headline-lg-mobile:
    fontFamily: Geist
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  headline-md:
    fontFamily: Geist
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  title-lg:
    fontFamily: Geist
    fontSize: 20px
    fontWeight: '600'
    lineHeight: 28px
  body-lg:
    fontFamily: Inter
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 28px
  body-md:
    fontFamily: Inter
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 24px
  body-sm:
    fontFamily: Inter
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 20px
  label-md:
    fontFamily: Geist
    fontSize: 14px
    fontWeight: '500'
    lineHeight: 20px
    letterSpacing: 0.05em
  label-sm:
    fontFamily: Geist
    fontSize: 12px
    fontWeight: '500'
    lineHeight: 16px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  base: 8px
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  2xl: 48px
  3xl: 64px
  container-max: 1280px
  gutter: 24px
  margin-mobile: 16px
---

## Brand & Style
The design system is engineered for high-stakes health insurance environments where precision, trust, and clarity are paramount. The brand personality is authoritative yet accessible, functioning as a reliable partner in complex decision-making processes.

The visual style follows a **Corporate / Modern** approach with a focus on data density and structural integrity. It avoids decorative trends like glassmorphism or heavy gradients in favor of "Evidence-Based Design"—where every element serves a functional purpose. The aesthetic is characterized by clean borders, generous whitespace to reduce cognitive load, and a strict adherence to a systematic grid.

## Colors
The palette is rooted in a "Deep Navy" primary to establish institutional trust. "Teal" is utilized for primary actions and progression indicators, providing a professional medical-adjacent feel. "Soft Saffron" is reserved strictly for high-priority accents and warnings to ensure immediate visual attention without causing alarm.

Surface colors utilize a "Soft Grey" background to reduce screen glare during long periods of use, with "White" surfaces used to elevate content containers. Neutral slates are used for typography to maintain high contrast ratios compliant with accessibility standards.

## Typography
The system employs a dual-font strategy: **Geist** for headings and technical labels to provide a precise, modern, and slightly technical feel; and **Inter** for body copy to ensure maximum legibility in data-rich environments.

Type scales are generous to accommodate medical terminology and complex policy descriptions. High-level headers use tighter letter spacing and heavier weights to command authority, while body text maintains standard tracking for flow.

## Layout & Spacing
This design system is built on a strict **8px grid system**. All margins, paddings, and component heights must be multiples of 8px to ensure mathematical harmony and visual stability.

The layout follows a **Fixed Grid** model for desktop (1280px max-width) to maintain readability of wide data tables. On mobile, the system transitions to a fluid single-column layout with 16px side margins. 

Internal spacing (padding) within cards and sections should default to `lg` (24px) to provide the "generous whitespace" required for a professional, un-cluttered interface.

## Elevation & Depth
Depth is conveyed primarily through **Tonal Layers** and **Low-Contrast Outlines**. 

- **Level 0 (Background):** Soft Grey (#f8fafc) - The canvas.
- **Level 1 (Surface):** White (#ffffff) - Cards and main content areas. These should use a subtle 1px border (#e2e8f0) rather than a shadow.
- **Level 2 (Popovers/Modals):** White (#ffffff) with a very soft, high-diffusion ambient shadow (0px 4px 20px rgba(15, 23, 42, 0.08)). 

The goal is a "flat-plus" look where hierarchy is established by structure and subtle containment rather than simulated physical distance.

## Shapes
The shape language is **Soft (0.25rem)**. This subtle rounding removes the harshness of sharp corners—making the professional interface feel approachable—without veering into the "playful" territory of highly rounded or pill-shaped designs. 

Standard components (buttons, inputs) use the base 4px radius. Larger containers (cards, modals) may use 8px (rounded-lg) to soften the overall layout.

## Components
- **Buttons:** Primary buttons use the Teal (#0d9488) fill with white text. Secondary buttons use a Deep Navy outline. Action buttons should have a minimum height of 40px (5x 8px units).
- **Inputs:** Standardize on a "White" fill with a 1px "Slate" border. Focus states must use a 2px Teal ring.
- **Cards:** Cards should be border-only (1px #e2e8f0) on the Soft Grey background. Insets should be 24px.
- **Chips/Badges:** Used for claim status. Use low-saturation background tints of the status colors (e.g., Success Green at 10% opacity) with high-saturation text.
- **Data Tables:** High-density with 1px horizontal dividers only. Header rows should be subtly shaded in Soft Grey with Geist Medium labels.
- **Progress Indicators:** Linear stepped progress bars using Teal to indicate completion of insurance claim stages.