import React from 'react';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import { entityTabValues, EntityTab } from './types';

type EntityTabToggleGroupProps = {
    entityContext?: EntityTab;
    cveCount?: number;
    imageCount?: number;
    deploymentCount?: number;
};

function EntityTabToggleGroup({
    entityContext,
    cveCount = 0,
    imageCount = 0,
    deploymentCount = 0,
}: EntityTabToggleGroupProps) {
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion(
        'entityTab',
        entityTabValues
    );

    return (
        <ToggleGroup className="pf-u-pl-md pf-u-pt-md">
            {entityContext !== 'CVE' && (
                <ToggleGroupItem
                    text={`${cveCount} CVEs`}
                    buttonId="cves"
                    isSelected={activeEntityTabKey === 'CVE'}
                    onChange={() => setActiveEntityTabKey('CVE')}
                />
            )}
            {entityContext !== 'Image' && (
                <ToggleGroupItem
                    text={`${imageCount} Images`}
                    buttonId="images"
                    isSelected={activeEntityTabKey === 'Image'}
                    onChange={() => setActiveEntityTabKey('Image')}
                />
            )}
            {entityContext !== 'Deployment' && (
                <ToggleGroupItem
                    text={`${deploymentCount} Deployments`}
                    buttonId="deployments"
                    isSelected={activeEntityTabKey === 'Deployment'}
                    onChange={() => setActiveEntityTabKey('Deployment')}
                />
            )}
        </ToggleGroup>
    );
}

export default EntityTabToggleGroup;
