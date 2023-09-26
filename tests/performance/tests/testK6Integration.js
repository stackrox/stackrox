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
