import { sleep } from 'k6';
import { getHeaderWithAdminPass, getHeaderWithToken } from '../src/utils.js';
import { createRng, randomBetween, pickWeighted } from '../src/random.js';
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

const pages = [
    { name: 'mainDashboard', fn: mainDashboard, weight: 1.0 },
    { name: 'violations', fn: violations, weight: 0.7 },
    { name: 'imageCveManagement', fn: imageCveManagement, weight: 0.5 },
    { name: 'vulnMgmtDashboard', fn: vulnerabilityManagementDashboard, weight: 0.5 },
    { name: 'configManagement', fn: configManagement, weight: 0.3 },
    { name: 'reports', fn: reports, weight: 0.2 },
    { name: 'deployments', fn: deployments, weight: 0.4 },
    { name: 'nodeCveManagement', fn: nodeCveManagement, weight: 0.3 },
    { name: 'clustersAndPolicies', fn: clustersAndPolicies, weight: 0.3 },
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
            exec: 'userJourney',
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
    },
});

function getHeaders() {
    if (__ENV.ROX_API_TOKEN) {
        return getHeaderWithToken(__ENV.ROX_API_TOKEN);
    }
    return getHeaderWithAdminPass(__ENV.ROX_ADMIN_PASSWORD);
}

export function userJourney() {
    const rng = createRng(__VU, __ITER);
    const headers = getHeaders();

    loginInit(__ENV.HOST, headers, { page: 'login' });
    sleep(randomBetween(rng, 1, 3));

    const selectedPages = pickWeighted(rng, pages);

    for (const page of selectedPages) {
        page.fn(__ENV.HOST, headers, { page: page.name });
        sleep(randomBetween(rng, 3, 10));
    }
}
