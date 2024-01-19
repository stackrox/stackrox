import React from 'react';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';

function CoverageTableViewToggleGroup() {
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion('tableView', [
        'Clusters',
        'Profiles',
    ]);

    function handleEntityTabChange(view) {
        setActiveEntityTabKey(view);
    }

    return (
        <ToggleGroup
            aria-label="Toggle for coverage view"
            className="pf-u-background-color-100 pf-u-p-lg"
        >
            <ToggleGroupItem
                text="Clusters"
                buttonId="compliance-clusters-toggle-group"
                isSelected={activeEntityTabKey === 'Clusters'}
                onChange={() => handleEntityTabChange('Clusters')}
            />
            <ToggleGroupItem
                text="Profiles (coming soon)"
                buttonId="compliance-profiles-toggle-group"
                isSelected={activeEntityTabKey === 'Profiles'}
                onChange={() => handleEntityTabChange('Profiles')}
                isDisabled
            />
        </ToggleGroup>
    );
}

export default CoverageTableViewToggleGroup;
