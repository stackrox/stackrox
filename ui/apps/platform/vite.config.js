import { defineConfig } from 'vite';

import react from '@vitejs/plugin-react';
import basicSsl from '@vitejs/plugin-basic-ssl';
import svgr from 'vite-plugin-svgr';

import { viteProxy } from './src/setupProxy.js';

function getEnv() {
    try {
        return Object.fromEntries(
            Object.entries(process.env).filter(([key]) => key.startsWith('VITE_'))
        );
    } catch {
        return {};
    }
}

export default defineConfig((props) => {
    const envWithProcessPrefix = {
        'process.env': getEnv(),
    };

    return {
        build: {
            outDir: 'build',
        },
        define: envWithProcessPrefix,
        plugins: [react(), basicSsl(), svgr()],
        resolve: {
            alias: {
                Components: '/src/Components',
                Containers: '/src/Containers',
                'app.tw.css': '/src/app.tw.css',
                configureApolloClient: '/src/configureApolloClient.js',
                constants: '/src/constants',
                css: '/src/css',
                'css.imports': '/src/css.imports.ts',
                'global/initializeAnalytics': '/src/global/initializeAnalytics',
                hooks: '/src/hooks',
                images: '/src/images',
                index: '/src/index.tsx',
                installRaven: '/src/installRaven.js',
                messages: '/src/messages',
                mockData: '/src/mockData',
                possibleTypes: '/src/possibleTypes.json',
                queries: '/src/queries',
                'react-app-env.d': '/src/react-app-env.d.ts',
                reducers: '/src/reducers',
                routePaths: '/src/routePaths.ts',
                sagas: '/src/sagas',
                services: '/src/services',
                setupProxy: '/src/setupProxy.js',
                setupTests: '/src/setupTests.js',
                sorters: '/src/sorters',
                'store/configureStore': '/src/store/configureStore',
                'test-utils': '/src/test-utils',
                types: '/src/types',
                utils: '/src/utils',
            },
        },
        server: {
            proxy: viteProxy(),
            port: 3000,
        },
    };
});
