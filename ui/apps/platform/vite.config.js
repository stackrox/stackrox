/* eslint-disable no-console */
import * as fs from 'fs';
import * as path from 'path';
import { defineConfig } from 'vite';
import { randomUUID } from 'crypto';

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

/*
 * Alias all top level directories and files under `/src/` so that we can import them in our code
 * via `import * from 'Components/SomeComponent`;`. Without these aliases we would need to do
 * something like `import * from '@/Components/SomeComponent';`, which
 * although cleaner, would require changing nearly every file in our code base.
 */
function getSrcAliases() {
    const aliases = {};

    fs.readdirSync(path.resolve(__dirname, 'src'), { withFileTypes: true }).forEach(({ name }) => {
        const alias = name.includes('.') ? name.split('.').slice(0, -1).join('.') : name;
        aliases[alias] = `/src/${name}`;
    });

    return aliases;
}

export default defineConfig(async () => {
    const sslOptions = getSslOptions();
    return {
        build: {
            assetsDir: './static',
            outDir: 'build',
            rollupOptions: {
                output: {
                    // Break the following dependencies into their own chunks to limit memory usage during build and decouple large
                    // dependencies from their first entry point in our app's pages.
                    manualChunks: {
                        d3: ['d3'],
                        lodash: ['lodash'],
                        redoc: [
                            'redoc',
                            '@redocly/ajv',
                            '@redocly/config',
                            '@redocly/openapi-core',
                        ],
                        react: ['react', 'react-dom'],
                        apollo: ['@apollo/client'],
                        patternfly: ['@patternfly/react-core', '@patternfly/react-styles'],
                    },
                },
            },
        },
        css: {
            devSourcemap: false,
        },
        // Any environment variable that we want passed to our application code must be explicitly defined below
        define: {
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
            'process.env.VITE_ROX_PRODUCT_BRANDING': JSON.stringify(
                process.env.VITE_ROX_PRODUCT_BRANDING
            ),
            // Define `global` here due to redoc's usage of this NodeJS module
            global: {},
        },
        plugins: [react(), svgr(), ...(sslOptions?.basicSsl ? [sslOptions.basicSsl()] : [])],
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
            reporters: [
                [
                    'junit',
                    {
                        outputFile: `./test-results/reports/junit-report-${randomUUID()}.xml`,
                        classNameTemplate: '{basename} {title}',
                        nameTemplate: '{basename} {title}',
                    },
                ],
                'verbose',
            ],
            sequence: {
                hooks: 'list',
            },
            setupFiles: 'src/setupTests.js',
        },
    };
});
