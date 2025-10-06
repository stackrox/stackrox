const path = require('path');

const cypressVite = require('cypress-vite');

// This is needed because cypress-axe will attempt to resolve the axe.min.js file internally using CommonJS require,
// which is not supported in Vite. Instead, we resolve the path to the file using CommonJS require in the cypress config
// and make the path available at test runtime when injecting the axe core.
const axeCorePath = require.resolve('axe-core/axe.min.js');

/*
 * The helper function intended to provide automatic code completion for configuration in many popular code editors
 * had subtle side-effect to cause some typescript-eslint/no-unsafe-return errors in unit test files.
 *
 * const { defineConfig } = require('cypress'); // eslint-disable-line import/no-extraneous-dependencies
 * module.exports = defineConfig({ … });
 */

module.exports = {
    chromeWebSecurity: false, // Browser options
    defaultCommandTimeout: 8000, // Timeouts options
    numTestsKeptInMemory: 0, // Global options
    requestTimeout: 20000, // Timeouts options
    video: true, // Videos options
    videoCompression: 32, // Videos options

    retries: {
        // Configure retry attempts for `cypress run`
        // Attempt a single retry for failed tests when run headless
        runMode: 1,
        // Configure retry attempts for `cypress open`
        openMode: 0,
    },

    e2e: {
        baseUrl: 'https://localhost:3000',
        viewportHeight: 850, // Viewport options
        viewportWidth: 1440, // Viewport options
        setupNodeEvents: (on, config) => {
            // eslint-disable-next-line no-param-reassign
            config.env.AXE_CORE_PATH = axeCorePath;
            on('task', {
                beforeSuite(spec) {
                    // eslint-disable-next-line no-console
                    console.log(`${new Date().toISOString()} running test suite: ${spec.name}\n`);
                    return null;
                },
                joinPaths(paths) {
                    return path.join(...paths);
                },
            });
            on('file:preprocessor', cypressVite.vitePreprocessor());
            return config;
        },
    },

    component: {
        devServer: {
            framework: 'react',
            bundler: 'vite',
        },
        viewportHeight: 600,
        viewportWidth: 800,
    },
};
