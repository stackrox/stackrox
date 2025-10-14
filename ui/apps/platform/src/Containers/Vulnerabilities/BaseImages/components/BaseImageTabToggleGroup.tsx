import React from 'react';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import type { BaseImageDetailTab } from '../types';

const baseImageTabValues = ['cves', 'images'] as const;

export type BaseImageTabToggleGroupProps = {
    onChange?: (tab: BaseImageDetailTab) => void;
};

function BaseImageTabToggleGroup({ onChange }: BaseImageTabToggleGroupProps) {
    const [activeTabKey, setActiveTabKey] = useURLStringUnion('tab', baseImageTabValues);

    function handleTabChange(tab: BaseImageDetailTab) {
        setActiveTabKey(tab);
        onChange?.(tab);
    }

    return (
        <ToggleGroup aria-label="Base image detail tabs">
            <ToggleGroupItem
                text="CVEs"
                buttonId="cves"
                isSelected={activeTabKey === 'cves'}
                onChange={() => handleTabChange('cves')}
            />
            <ToggleGroupItem
                text="Images"
                buttonId="images"
                isSelected={activeTabKey === 'images'}
                onChange={() => handleTabChange('images')}
            />
        </ToggleGroup>
    );
}

export default BaseImageTabToggleGroup;
