import React from 'react';
import { useHistory, useParams } from 'react-router-dom';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import { complianceEnhancedCoveragePath } from 'routePaths';

export type CoverageTableViewToggleGroupProps = {
    activeToggle: 'checks' | 'clusters';
};

function CoverageTableViewToggleGroup({ activeToggle }: CoverageTableViewToggleGroupProps) {
    const { profileName } = useParams();
    const history = useHistory();

    const handleToggleChange = (resultsView) => {
        history.push(`${complianceEnhancedCoveragePath}/profiles/${profileName}/${resultsView}`);
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

export default CoverageTableViewToggleGroup;
