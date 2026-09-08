import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vite";

export default defineConfig({
  base: "/assets/app/",
  plugins: [svelte()],
  build: {
    emptyOutDir: true,
    outDir: "../cmd/sharelock/assets/app",
  },
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:8080",
      "/assets": "http://127.0.0.1:8080",
    },
  },
});
