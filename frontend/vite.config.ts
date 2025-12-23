import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";
import { VitePWA } from "vite-plugin-pwa";
import { resolve } from "path";

const resolvedSiteUrl =
  process.env.VITE_SITE_URL ||
  process.env.SITE_URL ||
  "https://rexec.pipeops.io";
const resolvedAllowedOrigins =
  process.env.VITE_ALLOWED_ORIGINS || process.env.ALLOWED_ORIGINS || "";
process.env.VITE_SITE_URL = resolvedSiteUrl;
process.env.VITE_ALLOWED_ORIGINS = resolvedAllowedOrigins;

const siteUrl = process.env.VITE_SITE_URL;
const allowedOrigins = process.env.VITE_ALLOWED_ORIGINS || "";

export default defineConfig({
  define: {
    "import.meta.env.VITE_SITE_URL": JSON.stringify(siteUrl),
    "import.meta.env.VITE_ALLOWED_ORIGINS": JSON.stringify(allowedOrigins),
  },
  plugins: [
    svelte(),
    VitePWA({
      strategies: "injectManifest",
      srcDir: "src",
      filename: "sw.ts",
      registerType: "autoUpdate",
      includeAssets: [
        "favicon.svg",
        "og-image.svg",
        "robots.txt",
        "sitemap.xml",
        "pwa-96x96.png",
        "pwa-192x192.png",
        "pwa-512x512.png",
        "screenshot-desktop.png",
        "screenshot-mobile.png",
      ],
      manifest: {
        name: "Rexec - Terminal as a Service",
        short_name: "Rexec",
        description:
          "Launch secure Linux terminals instantly in your browser. No setup required.",
        start_url: "/",
        display: "standalone",
        background_color: "#0a0a0a",
        theme_color: "#00ff41",
        orientation: "any",
        icons: [
          {
            src: "/pwa-96x96.png",
            sizes: "96x96",
            type: "image/png",
          },
          {
            src: "/pwa-192x192.png",
            sizes: "192x192",
            type: "image/png",
          },
          {
            src: "/pwa-512x512.png",
            sizes: "512x512",
            type: "image/png",
          },
          {
            src: "/pwa-512x512.png",
            sizes: "512x512",
            type: "image/png",
            purpose: "maskable",
          },
        ],
        screenshots: [
          {
            src: "/screenshot-desktop.png",
            sizes: "1280x720",
            type: "image/png",
            form_factor: "wide",
            label: "Rexec Terminal Dashboard",
          },
          {
            src: "/screenshot-mobile.png",
            sizes: "390x844",
            type: "image/png",
            form_factor: "narrow",
            label: "Rexec Terminal on Mobile",
          },
        ],
        categories: ["developer", "utilities", "productivity"],
        lang: "en",
        dir: "ltr",
        scope: "/",
        shortcuts: [
          {
            name: "New Terminal",
            short_name: "New",
            description: "Create a new terminal session",
            url: "/ui/dashboard?action=create",
            icons: [
              { src: "/pwa-96x96.png", sizes: "96x96", type: "image/png" },
            ],
          },
          {
            name: "Dashboard",
            short_name: "Dashboard",
            description: "View your terminals",
            url: "/ui/dashboard",
            icons: [
              { src: "/pwa-96x96.png", sizes: "96x96", type: "image/png" },
            ],
          },
        ],
      },
      injectManifest: {
        globPatterns: ["**/*.{js,css,html,svg,png}"],
      },
      devOptions: {
        enabled: true,
        type: "module",
      },
    }),
  ],

  // Build output goes to Go's web directory
  build: {
    outDir: "../web",
    emptyOutDir: true,
    sourcemap: false,
    target: "esnext",
    minify: "terser",
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true,
      },
    },
    rollupOptions: {
      output: {
        // Code splitting for better caching
        entryFileNames: "assets/[name]-[hash].js",
        chunkFileNames: "assets/[name]-[hash].js",
        assetFileNames: "assets/[name]-[hash][extname]",
        manualChunks(id) {
          if (!id.includes("node_modules")) return;
          if (id.includes("@xterm/")) return "xterm";
          if (id.includes("/svelte/") || id.includes("node_modules/svelte"))
            return "svelte";
          return "vendor";
        },
      },
    },
  },

  resolve: {
    alias: {
      $lib: resolve("./src/lib"),
      $components: resolve("./src/lib/components"),
      $stores: resolve("./src/lib/stores"),
      $utils: resolve("./src/lib/utils"),
    },
  },

  // Dev server proxy to Go backend
  server: {
    port: 3000,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
      },
    },
  },
});
