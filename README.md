# sveltekit.fyi

Inspired by [nuxt.fyi](https://nuxt.fyi/), this is a project to scan Bluesky's firehose for sites made with Sveltekit.

## Detectors

### HTML based
- [x] [data-sveltekit-preload-data]
- [x] [data-sveltekit-keepalive]
- [x] script[data-sveltekit-async-loader]
- [x] [data-svelte-h^="svelte-"]
- [x] div#svelte-announcer

### JS based (unplanned)
- [ ] window.__svelte
- [ ] window.__sveltekit_{hash}
