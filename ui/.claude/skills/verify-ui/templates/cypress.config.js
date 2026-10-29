// Cypress config for verify-ui runs: the repo config plus a browser window large enough to
// hold the whole 1440x850 viewport next to the command log. Headless Electron defaults to
// 1280x720, which crops the app on the right.
const path = require('path');

const platformDir = process.env.VERIFY_PLATFORM_DIR;
const base = require(path.join(platformDir, 'cypress.config.js'));

const windowWidth = 2000;
const windowHeight = 1100;

module.exports = {
    ...base,
    e2e: {
        ...base.e2e,
        setupNodeEvents: (on, config) => {
            on('before:browser:launch', (browser, launchOptions) => {
                if (browser.name === 'electron') {
                    launchOptions.preferences.width = windowWidth;
                    launchOptions.preferences.height = windowHeight;
                } else {
                    launchOptions.args.push(`--window-size=${windowWidth},${windowHeight}`);
                }
                return launchOptions;
            });
            return base.e2e.setupNodeEvents(on, config);
        },
    },
};
