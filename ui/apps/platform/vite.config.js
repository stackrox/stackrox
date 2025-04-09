import * as fs from 'fs';
import * as path from 'path';
import { defineConfig } from 'vite';

import react from '@vitejs/plugin-react-swc';
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

function getSrcAliases() {
    const aliases = {};

    fs.readdirSync(path.resolve(__dirname, 'src'), { withFileTypes: true }).forEach(({ name }) => {
        const alias = name.includes('.') ? name.split('.').slice(0, -1).join('.') : name;
        aliases[alias] = `/src/${name}`;
    });

    return aliases;
}

export default defineConfig(async (params) => {
    const Inspect = (await import('vite-plugin-inspect')).default;
    const sslOptions = getSslOptions();
    return {
        build: {
            outDir: 'build',
            rollupOptions: {
                output: {
                    // Break the following dependencies into their own chunks to limit memory usage during build
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
                        // monaco: ['monaco-editor'],
                    },
                },
            },
        },
        css: {
            devSourcemap: false,
        },
        define: {
            'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
            'process.env.VITE_ROX_PRODUCT_BRANDING': JSON.stringify(
                process.env.VITE_ROX_PRODUCT_BRANDING
            ),
            // Define `global` here due to redoc's usage of this NodeJS module
            global: {},
        },
        optimizeDeps: {
            exclude: ['@apollo/client'],
        },
        plugins: [
            Inspect({
                build: true,
                outputDir: '/tmp/.vite-inspect',
            }),
            // Skip processing CSS in ./node_modules/ with PostCSS transforms
            {
                name: 'skip-postcss-node-modules',
                enforce: 'pre',
                transform(code, id) {
                    if (id.includes('node_modules') && id.endsWith('.css')) {
                        return {
                            code,
                            map: null,
                        };
                    }
                },
            },
            react(),
            svgr(),
            ...(sslOptions?.basicSsl ? [sslOptions.basicSsl()] : []),
        ],
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
