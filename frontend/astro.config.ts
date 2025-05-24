import { fileURLToPath } from 'node:url';

import { defineConfig } from 'astro/config';

import react from '@astrojs/react';
import node from '@astrojs/node';
import tailwind from '@astrojs/tailwind';
import icon from 'astro-icon';
// import VitePWA from '@vite-pwa/astro';

export default defineConfig({
  output: 'server',
  adapter: node({
    mode: 'standalone',
  }),

  server: {
    host: '0.0.0.0',
    port: 4321,
  },

  integrations: [
    react(),
    tailwind({
      applyBaseStyles: true,
    }),
    icon({
      include: {
        tabler: ['*'],
        'material-symbols': ['*'],
      },
    }),
    // 一時的にPWA機能を無効化
    // VitePWA({
    //   registerType: 'autoUpdate',
    //   workbox: {
    //     globPatterns: ['**/*.{js,css,html,ico,png,svg,webp,woff,woff2}'],
    //   },
    //   manifest: {
    //     name: 'Kasaneha - AI日記アプリ',
    //     short_name: 'Kasaneha',
    //     description: 'AIと対話して感情を分析する日記アプリ',
    //     theme_color: '#1f2937',
    //     background_color: '#f9fafb',
    //     display: 'standalone',
    //     orientation: 'portrait',
    //     scope: '/',
    //     start_url: '/',
    //     icons: [
    //       {
    //         src: '/icons/icon-192x192.png',
    //         sizes: '192x192',
    //         type: 'image/png',
    //       },
    //       {
    //         src: '/icons/icon-512x512.png',
    //         sizes: '512x512',
    //         type: 'image/png',
    //       },
    //     ],
    //   },
    // }),
  ],

  vite: {
    resolve: {
      alias: {
        '~': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    define: {
      'import.meta.env.PUBLIC_API_BASE_URL': JSON.stringify(
        process.env.PUBLIC_API_BASE_URL || 'http://localhost:8080'
      ),
    },
    server: {
      host: '0.0.0.0',
      port: 4321,
    },
  },
});
