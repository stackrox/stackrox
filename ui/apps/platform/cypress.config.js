/*
 * The helper function intended to provide automatic code completion for configuration in many popular code editors
 * had subtle side-effect to cause some typescript-eslint/no-unsafe-return errors in unit test files.
 *
 * const { defineConfig } = require('cypress'); // eslint-disable-line import/no-extraneous-dependencies
 * module.exports = defineConfig({ … });
 */

module.exports = {
    blockHosts: ['*.*'], // Browser options
    chromeWebSecurity: false, // Browser options
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
        specPattern: 'cypress/integration/**/*.test.{js,ts}',
        viewportHeight: 850, // Viewport options
        viewportWidth: 1440, // Viewport options
        setupNodeEvents: (on) => {
            on('task', {
                beforeSuite(spec) {
                    // eslint-disable-next-line no-console
                    console.log(`${new Date().toISOString()} running test suite: ${spec.name}\n`);
                    return null;
                },
            });
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
