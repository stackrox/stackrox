const path = require('path');

const testResultsDir = process.env.TEST_RESULTS_OUTPUT_DIR || 'cypress/test-results';
const threadId = process.env.CYPRESS_THREAD || '';
const threadSuffix = threadId ? `-thread${threadId}` : '';

module.exports = {
    reporterEnabled: 'spec, mocha-junit-reporter',
    mochaJunitReporterReporterOptions: {
        mochaFile: path.join(testResultsDir, `reports/[suiteFilename]${threadSuffix}-[hash].xml`),
        testCaseSwitchClassnameAndName: true,
    },
};
