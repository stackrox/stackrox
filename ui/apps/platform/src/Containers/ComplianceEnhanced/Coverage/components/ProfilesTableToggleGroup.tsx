import React from 'react';
import { useParams } from 'react-router-dom-v5-compat';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import {
    coverageProfileChecksPath,
    coverageProfileClustersPath,
} from '../compliance.coverage.routes';
import type { CoverageProfilePath } from '../compliance.coverage.routes';
import useScanConfigRouter from '../hooks/useScanConfigRouter';

export type ProfilesTableToggleGroupProps = {
    activeToggle: 'checks' | 'clusters';
};

function ProfilesTableToggleGroup({ activeToggle }: ProfilesTableToggleGroupProps) {
    const { navigateWithScanConfigQuery } = useScanConfigRouter();
    const { profileName } = useParams();
    const { searchFilter } = useURLSearch();

    const handleToggleChange = (resultsView) => {
        const path: CoverageProfilePath =
            resultsView === 'checks' ? coverageProfileChecksPath : coverageProfileClustersPath;
        navigateWithScanConfigQuery(path, { profileName }, { searchFilter });
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
