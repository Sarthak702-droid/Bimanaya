---
name: Clinical Integrity & Drafting
colors:
  surface: '#f8f9ff'
  surface-dim: '#cbdbf5'
  surface-bright: '#f8f9ff'
  surface-container-lowest: '#ffffff'
  surface-container-low: '#eff4ff'
  surface-container: '#e5eeff'
  surface-container-high: '#dce9ff'
  surface-container-highest: '#d3e4fe'
  on-surface: '#0b1c30'
  on-surface-variant: '#45464d'
  inverse-surface: '#213145'
  inverse-on-surface: '#eaf1ff'
  outline: '#76777d'
  outline-variant: '#c6c6cd'
  surface-tint: '#565e74'
  primary: '#000000'
  on-primary: '#ffffff'
  primary-container: '#131b2e'
  on-primary-container: '#7c839b'
  inverse-primary: '#bec6e0'
  secondary: '#0051d5'
  on-secondary: '#ffffff'
  secondary-container: '#316bf3'
  on-secondary-container: '#fefcff'
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
  secondary-fixed: '#dbe1ff'
  secondary-fixed-dim: '#b4c5ff'
  on-secondary-fixed: '#00174b'
  on-secondary-fixed-variant: '#003ea8'
  tertiary-fixed: '#ffddb8'
  tertiary-fixed-dim: '#ffb95f'
  on-tertiary-fixed: '#2a1700'
  on-tertiary-fixed-variant: '#653e00'
  background: '#f8f9ff'
  on-background: '#0b1c30'
  surface-variant: '#d3e4fe'
typography:
  display-doc:
    fontFamily: Work Sans
    fontSize: 36px
    fontWeight: '700'
    lineHeight: 44px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Work Sans
    fontSize: 28px
    fontWeight: '600'
    lineHeight: 36px
  headline-lg-mobile:
    fontFamily: Work Sans
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
  body-main:
    fontFamily: Source Serif 4
    fontSize: 18px
    fontWeight: '400'
    lineHeight: 30px
  body-compact:
    fontFamily: Source Serif 4
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 26px
  label-bold:
    fontFamily: Inter
    fontSize: 12px
    fontWeight: '600'
    lineHeight: 16px
    letterSpacing: 0.05em
  mono-ui:
    fontFamily: jetbrainsMono
    fontSize: 13px
    fontWeight: '400'
    lineHeight: 20px
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  container-max: 1280px
  editor-width: 720px
  sidebar-width: 320px
  gutter: 24px
  stack-sm: 8px
  stack-md: 16px
  stack-lg: 32px
---

## Brand & Style
The design system is engineered for high-stakes editorial review and legal grievance management. The brand personality is authoritative, meticulous, and objective. It balances "Clinical Integrity"—a state of formal, finished documentation—with a "Drafting" sub-theme that provides a safe, flexible workspace for composition and critique.

The visual style is **Corporate / Modern** with a lean toward **Minimalism**. It prioritizes high legibility and information density without overwhelming the user. The interface should feel like a precision instrument: stable, responsive, and intellectually supportive.

## Colors
The palette is anchored in "Deep Slate" (Primary) to establish authority and "Signal Blue" (Secondary) for interactive elements. 

- **Clinical Base:** Uses a high-contrast grayscale for maximum text clarity. Backgrounds are pure white (#FFFFFF) for documents and cool gray (#F8FAFC) for the workspace chrome.
- **Drafting Sub-theme:** Utilizes soft, purposeful highlights (Editor Highlights) to signify collaborative layers without vibrating against the text. 
- **Status Indicators:** A semantic system for grievance states: Amber for pending review, Emerald for verified integrity, and Slate for ongoing drafts.

## Typography
This design system employs a tri-font strategy to differentiate between structure, content, and data.

- **Headlines (Work Sans):** Professional and grounded. Used for navigation and document titles.
- **Body (Source Serif 4):** The core of the editorial experience. A transitional serif chosen for its exceptional readability in long-form grievances. Line height is set at a generous 1.6x to reduce eye strain during deep review.
- **Labels & UI (Inter):** Systematic and neutral. Used for metadata, button labels, and secondary information.
- **Data (JetBrains Mono):** Used for version timestamps and grievance ID tracking to provide a "technical audit" feel.

**Editorial Clarity Rule:** The main document body must maintain an optimal line length of 65–75 characters (approx. 720px max-width) to ensure reading speed and comprehension.

## Layout & Spacing
The layout follows a **Fixed-Fluid Hybrid** model. The workspace uses a 12-column grid, but the document editor is centered and capped at a fixed width for editorial focus.

- **Desktop:** A three-pane layout. Left: Navigation/Outlining. Center: The Editor (fixed 720px). Right: Review/Comments (320px).
- **Tablet:** Collapsible sidebars; the editor takes priority.
- **Mobile:** Single column; review comments move to an expandable bottom sheet.
- **Spacing Rhythm:** Uses an 8px base unit. Component internal padding is strictly 12px or 16px to maintain a dense but organized professional aesthetic.

## Elevation & Depth
The system uses **Tonal Layers** and **Low-contrast Outlines** rather than heavy shadows to maintain a "clinical" feel.

- **Level 0 (Background):** #F8FAFC. The workspace floor.
- **Level 1 (Document Surface):** White background with a 1px border (#E2E8F0). No shadow. This represents the "official record."
- **Level 2 (Review Overlays):** Floating comment bubbles or tooltips use a very soft ambient shadow (0px 4px 12px rgba(15, 23, 42, 0.05)) to distinguish them from the text without breaking the flat editorial plane.
- **Interactions:** Hover states on interactive rows use a subtle tint (#F1F5F9) rather than an elevation lift.

## Shapes
The shape language is **Soft** and restrained. 
- **Standard Elements:** 4px (0.25rem) radius for buttons, input fields, and cards. This sharp-but-not-harsh corner reinforces the "precise" and "institutional" nature of the work.
- **Badges/Tags:** 2px radius or sharp to denote formal categorization.
- **Comment Bubbles:** Use a slightly larger 8px (0.5rem) radius to differentiate "human conversation" from "document data."

## Components
Consistent styling for the Reviewer Workspace:

- **Editor Highlights:** Use background-color spans for text selection. `editor_highlights.comment` (Blue) for general notes, `editor_highlights.suggestion` (Green) for proposed edits.
- **Reviewer Signatures:** A specialized component at the footer of documents. Includes a digital "Verified" badge (Emerald), a scanned signature placeholder, and the professional credentials in `label-bold`.
- **Comment Bubbles:** Positioned in the right margin. Must include a timestamp in `mono-ui` and an avatar with a status ring indicating the reviewer's seniority level.
- **Trust Badges:** Small, high-contrast labels (e.g., "LEGAL REVIEWED", "CLINICAL AUDIT") using `label-bold` with 1px letter spacing.
- **Action Buttons:** Primary actions are solid #0F172A; secondary actions use "Signal Blue" outlines.
- **Input Fields:** Minimalist design with only a bottom border (2px) in active state to mimic a physical form.
- **Status Indicators:** Circular dots paired with `label-bold` text for grievance lifecycle tracking.