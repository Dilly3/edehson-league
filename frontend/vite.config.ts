import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import type { Plugin } from "vite";

// Each build gets a unique version. It is compiled into the app and written to
// /version.json, so open tabs can notice a new deploy and offer to refresh.
const appVersion = new Date().toISOString();
function versionFile(): Plugin {
  return {
    name: "app-version-file",
    apply: "build",
    generateBundle() {
      this.emitFile({ type: "asset", fileName: "version.json", source: JSON.stringify({ version: appVersion }) });
    },
  };
}
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const proxy = {
    "/api": {
      target: env.API_PROXY_TARGET || "http://127.0.0.1:8011",
      changeOrigin: true,
    },
  };
  return {
    plugins: [react(), versionFile()],
    define: { __APP_VERSION__: JSON.stringify(appVersion) },
    server: { port: 5173, strictPort: true, proxy },
    preview: { port: 4173, proxy },
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            ui: ["@mui/material", "@emotion/react", "@emotion/styled"],
          },
        },
      },
    },
  };
});
