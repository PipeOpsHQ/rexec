import { defineConfig } from "vite";
import { resolve } from "path";

export default defineConfig({
  build: {
    lib: {
      entry: resolve(__dirname, "src/index.ts"),
      name: "Rexec",
      formats: ["umd", "es"],
      fileName: (format) => {
        if (format === "umd") return "rexec.min.js";
        if (format === "es") return "rexec.esm.js";
        return `rexec.${format}.js`;
      },
    },
    outDir: "dist",
    emptyOutDir: true,
    minify: "esbuild",
    rollupOptions: {
      output: {
        // Use named exports to avoid the default export warning
        exports: "named",
        // Ensure CSS is inlined
        inlineDynamicImports: true,
        // Global variable name for UMD build
        name: "Rexec",
        // Ensure all assets are bundled
        assetFileNames: "rexec.[ext]",
      },
    },
    // Inline all CSS into JS - cssCodeSplit false alone doesn't work for libraries
    cssCodeSplit: false,
    // Target modern browsers but keep compatibility
    target: "es2018",
    // Generate source maps for debugging
    sourcemap: true,
  },
  // Force CSS to be injected into JS at runtime
  css: {
    // Inject CSS into the document head at runtime
    devSourcemap: true,
  },
  plugins: [
    // Custom plugin to inject CSS into JS bundle
    {
      name: "css-inject",
      apply: "build",
      enforce: "post",
      generateBundle(options, bundle) {
        // Find the CSS file and JS files
        let cssContent = "";
        const cssFiles: string[] = [];

        for (const [fileName, chunk] of Object.entries(bundle)) {
          if (fileName.endsWith(".css") && chunk.type === "asset") {
            cssContent = chunk.source as string;
            cssFiles.push(fileName);
          }
        }

        if (cssContent) {
          // Remove CSS files from bundle
          for (const cssFile of cssFiles) {
            delete bundle[cssFile];
          }

          // Inject CSS into JS files
          const cssInjector = `(function(){if(typeof document!=="undefined"){var s=document.createElement("style");s.textContent=${JSON.stringify(cssContent)};document.head.appendChild(s);}})();`;

          for (const [fileName, chunk] of Object.entries(bundle)) {
            if (chunk.type === "chunk" && fileName.endsWith(".js")) {
              chunk.code = cssInjector + chunk.code;
            }
          }
        }
      },
    },
  ],
  resolve: {
    alias: {
      "@": resolve(__dirname, "src"),
    },
  },
  define: {
    // Prevent issues with process.env in browser
    "process.env.NODE_ENV": JSON.stringify("production"),
  },
});
