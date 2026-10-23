/**
 * Guards the CSS classes our layout depends on.
 *
 * daisyUI 5 silently dropped `.form-control`, `.input-bordered` and
 * `.select-bordered`; the classes kept being emitted in JSX but produced no CSS.
 * `.form-control` was the costly one: without its column layout the field
 * wrappers stayed inline, so the `space-y-*` margins between fields were
 * ignored on inline boxes and the submit button ended up flush against the last
 * input.
 *
 * This check fails the build when a class listed below is missing from the
 * compiled stylesheet, so the next silent removal in a dependency upgrade shows
 * up as a red build instead of a layout bug. If a class vanishes because its
 * last usage was removed, delete it from this list too.
 */
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'

const REQUIRED_CLASSES = [
  // our own component classes (index.css)
  'surface',
  // form fields and their wrappers
  'form-control',
  'input',
  'select',
  'toggle',
  // buttons and button groups
  'btn',
  'join',
  'join-item',
  // overlays
  'modal',
  'modal-box',
  'modal-action',
  'dropdown',
  // feedback
  'alert',
  'loading',
  // layout primitives
  'rounded-box',
  'menu',
  'table',
]

const cssDir = fileURLToPath(new URL('../dist/assets/', import.meta.url))
const files = [...new Bun.Glob('*.css').scanSync({ cwd: cssDir })]
if (files.length === 0) {
  throw new Error('No compiled CSS in frontend/dist — run the build before this check.')
}

const css = (await Promise.all(files.map((file) => Bun.file(join(cssDir, file)).text()))).join('\n')

const missing = REQUIRED_CLASSES.filter((name) => !new RegExp(`\\.${name}[^a-zA-Z0-9_-]`).test(css))

if (missing.length > 0) {
  console.error(`Missing CSS for classes the UI depends on: ${missing.join(', ')}`)
  process.exit(1)
}

// `.form-control` is ours (index.css): it must lay the field out as a column so
// the wrapper is a block-level box and `space-y-*` spacing between fields holds.
const formControl = css.match(/\.form-control\{([^}]*)\}/)?.[1] ?? ''
const formControlBroken = ['display:flex', 'flex-direction:column'].filter((rule) => !formControl.includes(rule))

if (formControlBroken.length > 0) {
  console.error(`.form-control lost ${formControlBroken.join(' and ')} — stacked fields lose their spacing.`)
  process.exit(1)
}

console.log(`UI classes OK (${REQUIRED_CLASSES.length} checked in ${files.join(', ')}).`)
