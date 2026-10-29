// Cypress config for verify-ui runs: the repo config plus a browser window large enough to
// hold the whole 1440x850 viewport next to the command log (headless Electron defaults to
// 1280x720, which crops the app on the right), the evidence dir exposed to specs, and a
// support file that adds the proof checks drive.sh gates on.
const path = require('path');

const platformDir = process.env.VERIFY_PLATFORM_DIR;
const base = require(path.join(platformDir, 'cypress.config.js'));

const windowWidth = 2000;
const windowHeight = 1100;

module.exports = {
    ...base,
    e2e: {
        ...base.e2e,
        supportFile: path.join(__dirname, 'support.js'),
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
            const result = base.e2e.setupNodeEvents(on, config) || config;
            // The repo config sets allowCypressEnv: false, so specs read public values through
            // Cypress.expose(). The evidence dir is a local path, not a secret.
            result.expose = {
                ...result.expose,
                VERIFY_EVIDENCE_DIR: result.env.VERIFY_EVIDENCE_DIR,
            };
            return result;
        },
    },
};
