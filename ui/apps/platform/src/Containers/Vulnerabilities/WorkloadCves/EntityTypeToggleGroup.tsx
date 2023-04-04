import React from 'react';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import { NonEmptyArray } from 'utils/type.utils';
import { entityTabValues, EntityTab } from './types';

type EntityTabToggleGroupProps = {
    className?: string;
    entityTabs?: Readonly<NonEmptyArray<EntityTab>>;
    cveCount?: number;
    imageCount?: number;
    deploymentCount?: number;
};

function EntityTabToggleGroup({
    className = '',
    entityTabs = entityTabValues,
    cveCount = 0,
    imageCount = 0,
    deploymentCount = 0,
}: EntityTabToggleGroupProps) {
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion('entityTab', entityTabs);

    return (
        <ToggleGroup className={className}>
            {entityTabs.includes('CVE') ? (
                <ToggleGroupItem
                    text={`${cveCount} CVEs`}
                    buttonId="cves"
                    isSelected={activeEntityTabKey === 'CVE'}
                    onChange={() => setActiveEntityTabKey('CVE')}
                />
            ) : (
                <></>
            )}
            {entityTabs.includes('Image') ? (
                <ToggleGroupItem
                    text={`${imageCount} Images`}
                    buttonId="images"
                    isSelected={activeEntityTabKey === 'Image'}
                    onChange={() => setActiveEntityTabKey('Image')}
                />
            ) : (
                <></>
            )}
            {entityTabs.includes('Deployment') ? (
                <ToggleGroupItem
                    text={`${deploymentCount} Deployments`}
                    buttonId="deployments"
                    isSelected={activeEntityTabKey === 'Deployment'}
                    onChange={() => setActiveEntityTabKey('Deployment')}
                />
            ) : (
                <></>
            )}
        </ToggleGroup>
    );
}

export default EntityTabToggleGroup;
