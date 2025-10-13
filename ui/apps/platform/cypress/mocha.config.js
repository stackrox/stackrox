const path = require('path');

const testResultsDir = process.env.TEST_RESULTS_OUTPUT_DIR || 'cypress/test-results';

module.exports = {
    reporterEnabled: 'spec, mocha-junit-reporter',
    mochaJunitReporterReporterOptions: {
        mochaFile: path.join(testResultsDir, 'reports/[suiteFilename]-results.xml'),
        testCaseSwitchClassnameAndName: true,
    },
};
