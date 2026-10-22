import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const proxy = {
    "/api": {
      target: env.API_PROXY_TARGET || "http://127.0.0.1:8011",
      changeOrigin: true,
    },
  };
  return {
    plugins: [react()],
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
