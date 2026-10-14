# Frontend

## Frontend rules

- Use the generated OpenAPI client in `frontend/src/shared/api/openapi.gen.ts`;
  never edit it manually. Run the generation script through the npm commands.
- WebTUI and the Catppuccin Macchiato theme are bundled by Vite. Do not add
  manually vendored WebTUI CSS or remote font/CDN URLs.
- Prefer WebTUI declarative layout and components. Keep custom CSS narrow,
  scoped, and based on `ch`/`lh` geometry where possible.
- Build assets with `npm --prefix frontend run build`; this updates
  `cmd/sharelock/assets/app`. Do not edit generated files there by hand.

## Interface languages

- The language selector exposes UA and EN and highlights the current interface language. Without a remembered manual choice, use the primary browser language: `uk` and `ru` (including regional variants) select Ukrainian; other or unavailable languages select English. Either selection is remembered, including clicking the already-active detected language. Previously saved Automatic preferences still use detection.
- Keep application-owned messages in `frontend/src/shared/i18n/messages.ts`, including whole sentences and Ukrainian count forms. Switching language must preserve drafts, generated links, and opened shares without making another open request.
- Display server-provided error details verbatim. Use `LocalizedError` for application validation and API fallbacks so existing notices follow language changes; unexpected browser failures receive a localized contextual fallback.
- The create form handles numeric validation in its submit path instead of native validation bubbles, so these messages follow the selected interface language. Operating-system dialogs remain outside application translation coverage.
