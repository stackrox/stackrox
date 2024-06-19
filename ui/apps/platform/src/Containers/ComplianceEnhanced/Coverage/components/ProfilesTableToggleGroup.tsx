import React from 'react';
import { useParams, useLocation } from 'react-router-dom';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';
import qs from 'qs';

import {
    coverageProfileChecksPath,
    coverageProfileClustersPath,
    CoverageProfilePath,
} from '../compliance.coverage.routes';
import useScanConfigRouter from '../hooks/useScanConfigRouter';

export type ProfilesTableToggleGroupProps = {
    activeToggle: 'checks' | 'clusters';
};

function ProfilesTableToggleGroup({ activeToggle }: ProfilesTableToggleGroupProps) {
    const { navigateWithScanConfigQuery } = useScanConfigRouter();
    const { profileName } = useParams();
    const location = useLocation();

    const handleToggleChange = (resultsView) => {
        const searchParams = qs.parse(location.search, { ignoreQueryPrefix: true });
        const path: CoverageProfilePath =
            resultsView === 'checks' ? coverageProfileChecksPath : coverageProfileClustersPath;
        navigateWithScanConfigQuery(path, { profileName }, searchParams);
    };

    return (
        <ToggleGroup aria-label="Toggle for coverage view">
            <ToggleGroupItem
                text="Checks"
                buttonId="compliance-clusters-toggle-group"
                isSelected={activeToggle === 'checks'}
                onChange={() => handleToggleChange('checks')}
            />
            <ToggleGroupItem
                text="Clusters"
                buttonId="compliance-clusters-toggle-group"
                isSelected={activeToggle === 'clusters'}
                onChange={() => handleToggleChange('clusters')}
            />
        </ToggleGroup>
    );
}

export default ProfilesTableToggleGroup;
