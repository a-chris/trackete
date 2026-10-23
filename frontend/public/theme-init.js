// Applies the saved theme before first paint so the app does not flash.
// Kept dependency-free and external so it satisfies `script-src 'self'`.
// Storage can throw (private mode, blocked cookies, sandboxed iframe), so every
// access is wrapped: a failure falls back to prefers-color-scheme.
;(function () {
  var stored = null
  try {
    stored = window.localStorage.getItem('trackete.theme')
  } catch (error) {
    stored = null
  }

  var mode = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system'

  var prefersDark = false
  try {
    prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  } catch (error) {
    prefersDark = false
  }

  var dark = mode === 'dark' || (mode === 'system' && prefersDark)
  document.documentElement.dataset.theme = dark ? 'trackete-dark' : 'trackete-light'
  document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
})()
