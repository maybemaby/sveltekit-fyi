# sveltekit.fyi

Inspired by [nuxt.fyi](https://nuxt.fyi/), this is a project to scan Bluesky's firehose for sites made with Sveltekit.

## Development

Clone the repo and setup a .env file including S3 credentials.

```bash
# run migrations
mise r migrate:up
# Run backend and frontend dev servers
mise r dev
```

### Requirements
- Mise (just runs tasks, not used to manage environment right now)
- Node 22+
- Go 1.25.7+


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
