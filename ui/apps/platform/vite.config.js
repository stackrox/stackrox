/* eslint-disable no-console */
import * as fs from 'fs';
import * as path from 'path';
import { defineConfig } from 'vite';

import react from '@vitejs/plugin-react';
import basicSsl from '@vitejs/plugin-basic-ssl';
import svgr from 'vite-plugin-svgr';

import { viteProxy } from './src/setupProxy';

function getSslOptions() {
    // When component testing, do not use SSL at all or the test executor will hang
    if (process.env.CYPRESS_COMPONENT_TEST) {
        console.log('Running Cypress component tests - SSL is disabled');
        return undefined;
    }

    // If local certs are defined, use them
    if (process.env.SSL_CRT_FILE && process.env.SSL_KEY_FILE) {
        console.log('Local certificate env vars detected');
        console.log(
            `Using SSL_CRT_FILE=${process.env.SSL_CRT_FILE} and SSL_KEY_FILE=${process.env.SSL_KEY_FILE} for local SSL`
        );
        return {
            localHttpsConfig: {
                https: {
                    cert: fs.readFileSync(process.env.SSL_CRT_FILE),
                    key: fs.readFileSync(process.env.SSL_KEY_FILE),
                },
            },
        };
    }

    // If certs are not defined and we are in development mode, fall back to basic SSL
    if (process.env.NODE_ENV === 'development') {
        console.warn('Falling back to basic SSL for development server');
        console.warn(
            'It is recommended to generate local certificates for secure development instead'
        );
        return { basicSsl };
    }

    return undefined;
}

function getEnv() {
    try {
        return Object.fromEntries(
            Object.entries(process.env).filter(([key]) => key.startsWith('VITE_'))
        );
    } catch {
        return {};
    }
}

function getSrcAliases() {
    const aliases = {};

    fs.readdirSync(path.resolve(__dirname, 'src'), { withFileTypes: true }).forEach(({ name }) => {
        const alias = name.includes('.') ? name.split('.').slice(0, -1).join('.') : name;
        aliases[alias] = `/src/${name}`;
    });

    return aliases;
}

export default defineConfig((params) => {
    const sslOptions = getSslOptions();
    return {
        build: {
            outDir: 'build',
        },
        define: {
            'process.env': getEnv(),
            // Define `global` here due to redoc's usage of this NodeJS module
            global: {},
        },
        plugins: [react(), svgr(), sslOptions?.basicSsl?.()],
        resolve: {
            alias: getSrcAliases(),
        },
        server: {
            proxy: viteProxy(),
            port: 3000,
            ...(sslOptions?.localHttpsConfig ?? {}),
        },
        test: {
            environment: 'jsdom',
            globals: true,
            reporters: ['junit', 'verbose'],
            outputFile: {
                junit: './test-results/reports/junit-report.xml',
            },
            sequence: {
                hooks: 'list',
            },
            setupFiles: 'src/setupTests.js',
        },
    };
});
