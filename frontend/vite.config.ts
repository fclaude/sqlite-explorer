import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({mode}) => ({
    plugins: mode === 'test' ? [] : [react()],
    esbuild: {
        jsx: 'automatic',
    },
    test: {
        environment: 'jsdom',
        setupFiles: './src/test/setup.ts',
        globals: true,
    },
}));
