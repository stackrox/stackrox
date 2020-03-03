// You can read more here:
// https://on.cypress.io/plugins-guide

// `on` is used to hook into various events Cypress emits
// `config` is the resolved Cypress config
module.exports = (on, config) => {
    if (config.env.COLLECT_COVERAGE) {
        // eslint-disable-next-line global-require
        on('task', require('@cypress/code-coverage/task'));
    }
};
