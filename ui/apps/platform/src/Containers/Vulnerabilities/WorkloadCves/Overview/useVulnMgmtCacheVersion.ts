import { useEffect, useState } from 'react';

import { subscribeVulnMgmtCache } from 'services/VulnMgmtService';

// Returns a counter that changes whenever the vulnerability management REST cache is
// invalidated, so data hooks can refetch the affected lists.
export function useVulnMgmtCacheVersion(): number {
    const [version, setVersion] = useState(0);
    useEffect(() => subscribeVulnMgmtCache(() => setVersion((current) => current + 1)), []);
    return version;
}
