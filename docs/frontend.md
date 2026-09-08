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
