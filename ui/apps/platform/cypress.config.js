const { defineConfig } = require('cypress'); // eslint-disable-line import/no-extraneous-dependencies

module.exports = defineConfig({
    blockHosts: ['*.*'], // Browser options
    chromeWebSecurity: false, // Browser options
    numTestsKeptInMemory: 0, // Global options
    viewportHeight: 850, // Viewport options
    viewportWidth: 1440, // Viewport options

    e2e: {
        baseUrl: 'https://localhost:3000',
        specPattern: 'cypress/integration/**/*.test.js',
    },
});
