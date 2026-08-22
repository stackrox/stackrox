import { sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';
import { jUnit, textSummary } from 'https://jslib.k6.io/k6-summary/0.0.2/index.js';
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/main/dist/bundle.js';
import { getHeaderWithAdminPass, getHeaderWithToken } from '../src/utils.js';
import { defaultOptions } from '../src/options.js';

import { loginInit } from '../groups/loginInit.js';
import { mainDashboard } from '../groups/mainDashboard.js';
import { violations } from '../groups/violations.js';
import { imageCveManagement } from '../groups/imageCveManagement.js';
import { nodeCveManagement } from '../groups/nodeCveManagement.js';
import { reports } from '../groups/reports.js';
import { vulnerabilityManagementDashboard } from '../groups/vulnerabilityManagementDashboard.js';
import { configManagement } from '../groups/configManagement.js';
import { deployments } from '../groups/deployments.js';
import { clustersAndPolicies } from '../groups/clustersAndPolicies.js';
import { collections } from '../groups/collections.js';
import { systemAdmin } from '../groups/systemAdmin.js';
import { compliance } from '../groups/compliance.js';
import { networkGraph } from '../groups/networkGraph.js';
import { risk } from '../groups/risk.js';
import { accessControl } from '../groups/accessControl.js';
import { integrations } from '../groups/integrations.js';
import { platformCveManagement } from '../groups/platformCveManagement.js';

const pages = [
    { name: 'mainDashboard', fn: mainDashboard, weight: 1.0 },
    { name: 'violations', fn: violations, weight: 0.7 },
    { name: 'imageCveManagement', fn: imageCveManagement, weight: 0.5 },
    { name: 'vulnMgmtDashboard', fn: vulnerabilityManagementDashboard, weight: 0.5 },
    { name: 'risk', fn: risk, weight: 0.4 },
    { name: 'deployments', fn: deployments, weight: 0.4 },
    { name: 'compliance', fn: compliance, weight: 0.35 },
    { name: 'configManagement', fn: configManagement, weight: 0.3 },
    { name: 'nodeCveManagement', fn: nodeCveManagement, weight: 0.3 },
    { name: 'clustersAndPolicies', fn: clustersAndPolicies, weight: 0.3 },
    { name: 'platformCveManagement', fn: platformCveManagement, weight: 0.25 },
    { name: 'networkGraph', fn: networkGraph, weight: 0.25 },
    { name: 'integrations', fn: integrations, weight: 0.2 },
    { name: 'reports', fn: reports, weight: 0.2 },
    { name: 'accessControl', fn: accessControl, weight: 0.15 },
    { name: 'collections', fn: collections, weight: 0.15 },
    { name: 'systemAdmin', fn: systemAdmin, weight: 0.1 },
];

export const options = Object.assign({}, defaultOptions, {
    scenarios: {
        user_journey: {
            executor: 'ramping-vus',
            startVUs: 1,
            stages: [
                { duration: '1m', target: 5 },
                { duration: '3m', target: 5 },
                { duration: '1m', target: 10 },
                { duration: '3m', target: 10 },
                { duration: '1m', target: 0 },
            ],
            exec: 'default',
        },
    },
    thresholds: {
        'http_req_duration{lib:true}': ['max>=0'],
        http_req_duration: ['p(95)<5000'],
        'group_duration{group:::login init}': ['p(95)<3000'],
        'group_duration{group:::main dashboard}': ['p(95)<5000'],
        'group_duration{group:::violations}': ['p(95)<5000'],
        'group_duration{group:::image cve management}': ['p(95)<5000'],
        'group_duration{group:::vulnerability management dashboard}': ['p(95)<10000'],
        'group_duration{group:::configuration management}': ['p(95)<20000'],
        http_req_failed: ['rate<0.01'],
    },
});

function getHeaders() {
    if (__ENV.ROX_API_TOKEN) {
        return getHeaderWithToken(__ENV.ROX_API_TOKEN);
    }
    return getHeaderWithAdminPass(__ENV.ROX_ADMIN_PASSWORD);
}

export function handleSummary(data) {
    return {
        stdout: textSummary(data, { indent: '  ', enableColors: true }),
        'performance-results/user-journey-report.txt': textSummary(data, { indent: '  ', enableColors: false }),
        'performance-results/user-journey-report.xml': jUnit(data),
        'performance-results/user-journey-report.json': JSON.stringify(data),
        'performance-results/user-journey-report.html': htmlReport(data),
    };
}

export default function userJourney() {
    const headers = getHeaders();

    loginInit(__ENV.HOST, headers, { page: 'login' });
    sleep(randomIntBetween(1, 3));

    const selected = pages.filter((p) => Math.random() < p.weight);
    for (let i = selected.length - 1; i > 0; i--) {
        const j = Math.floor(Math.random() * (i + 1));
        [selected[i], selected[j]] = [selected[j], selected[i]];
    }

    for (const page of selected) {
        page.fn(__ENV.HOST, headers, { page: page.name });
        sleep(randomIntBetween(3, 10));
    }
}
