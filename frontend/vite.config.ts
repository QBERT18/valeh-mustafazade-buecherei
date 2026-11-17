import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // The app always calls /api/...; in dev Vite forwards it to the Go server
  // (in the container nginx does the same), so the backend needs no CORS.
  server: {
    proxy: {
      "/api": { target: "http://localhost:8080", rewrite: (p) => p.replace(/^\/api/, "") },
    },
  },
});
