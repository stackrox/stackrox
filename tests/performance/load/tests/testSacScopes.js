import { getHeaderWithAdminPass, getHeaderWithToken } from '../src/utils.js';
import { createSac, deleteSac } from '../src/sacUtils.js';
import { vulnerabilityManagementDashboard } from '../groups/vulnerabilityManagementDashboard.js';
import { mainDashboard } from '../groups/mainDashboard.js';

import { defaultOptions } from '../src/options.js';

// SAC scope defintions.
const sacOptions = [
    {
        name: 'TestOneClusterOneNamespace',
        tag: 'c1-n1',
        numClusters: 1,
        numNamespaces: 1,
    },
    {
        name: 'TestFiveClustersFiveNamespaces',
        tag: 'c5-n5',
        numClusters: 5,
        numNamespaces: 5,
    },
    {
        name: 'TestTenClustersTenNamespaces',
        tag: 'c10-n10',
        numClusters: 10,
        numNamespaces: 10,
    },
    {
        name: 'TestTenClustersHundredNamespaces',
        tag: 'c10-n100',
        numClusters: 10,
        numNamespaces: 100,
    },
];

// k6 options.
export const options = defaultOptions;

export function setup() {
    const sacInfos = {};
    sacOptions.forEach((sacOption) => {
        const sacInfo = createSac(
            __ENV.HOST,
            getHeaderWithAdminPass(__ENV.ROX_ADMIN_PASSWORD),
            sacOption.name,
            sacOption.numClusters,
            sacOption.numNamespaces
        );

        sacInfos[sacOption.name] = sacInfo;
    });

    return sacInfos;
}

export function teardown(sacInfos) {
    sacOptions.forEach((sacOption) => {
        deleteSac(
            __ENV.HOST,
            getHeaderWithAdminPass(__ENV.ROX_ADMIN_PASSWORD),
            sacInfos[sacOption.name],
            sacOption.name
        );
    });
}

const runAllGroups = (header, tags) => {
    vulnerabilityManagementDashboard(__ENV.HOST, header, tags);
    mainDashboard(__ENV.HOST, header, tags);
};

export default function main(sacInfos) {
    // Run all with admin.
    console.log('Running tests for admin scope');
    runAllGroups(getHeaderWithAdminPass(__ENV.ROX_ADMIN_PASSWORD), { sac_user: 'admin' });

    // Run all groups for different scopes.
    sacOptions.forEach((sacOption) => {
        console.log('Running tests for scope', sacOption);

        runAllGroups(getHeaderWithToken(sacInfos[sacOption.name]['token']), {
            sac_user: sacOption.tag,
        });
    });
}
