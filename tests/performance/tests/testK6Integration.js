// TODO: these imports may be done in package.json
import { jUnit, textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';
import { htmlReport } from "https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js";

import { getHeaderWithAdminPass } from '../src/utils.js';
import { mainDashboard } from '../groups/mainDashboard.js';

import { defaultOptions } from '../src/options.js';

// k6 options.
export const options = defaultOptions;

const runAllGroups = (header, tags) => {
    mainDashboard(__ENV.HOST, header, tags);
};

export default function main() {
    // Run all with admin.
    console.log('Running tests for admin scope');
    runAllGroups(getHeaderWithAdminPass(__ENV.ROX_ADMIN_PASSWORD), { sac_user: 'admin' });
}

export function handleSummary(data) {
    return {
      'stdout': textSummary(data, { indent: '  ', enableColors: true }), // the default text output to stdout
      'performance-results/report.txt': textSummary(data, { indent: '  ', enableColors: false }), // the default text output to a file
      'performance-results/report.xml': jUnit(data), // JUnit output to a file
      'performance-results/report.json': JSON.stringify(data), // JSON output to a file
      'performance-results/report.html': htmlReport(data), // HTML report
    };
  }
