// Applies the saved theme before first paint so the app does not flash.
// Kept dependency-free and external so it satisfies `script-src 'self'`.
;(function () {
  try {
    var stored = window.localStorage.getItem('trackete.theme')
    var mode = stored === 'light' || stored === 'dark' || stored === 'system' ? stored : 'system'
    var dark =
      mode === 'dark' ||
      (mode === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches)
    document.documentElement.dataset.theme = dark ? 'trackete-dark' : 'trackete-light'
    document.documentElement.style.colorScheme = dark ? 'dark' : 'light'
  } catch (error) {
    // Storage or matchMedia unavailable: keep the markup default.
  }
})()
