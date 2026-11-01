import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// Toy assets are served from /toy/<slug>/; never assume the origin root.
export default defineConfig({
 resolve: {dedupe: ['vue', 'pinia', 'naive-ui', '@phosphor-icons/vue', 'seemly']},
  plugins: [vue()],
  base: './',
  build: { target: 'es2020', outDir: 'dist', assetsInlineLimit: 4096 },
  server: { host: '127.0.0.1', port:5173, strictPort:true, fs:{allow:['..']} },
})
