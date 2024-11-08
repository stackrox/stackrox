import * as fs from 'fs';
import { defineConfig } from 'vite';

import react from '@vitejs/plugin-react';
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
    return {
        build: {
            outDir: 'build',
        },
        define: {
            'process.env': getEnv(),
            // Define `global` here due to redoc's usage of this NodeJS module
            global: {},
        },
        plugins: [react(), svgr()],
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
            // TODO Discuss with team - does everyone have a self-signed cert?
            https: {
                key: fs.readFileSync('/Users/dvail/certs/DVailRootCA.pem'),
                cert: fs.readFileSync('/Users/dvail/certs/DVailRootCA.crt'),
            },
        },
    };
});
