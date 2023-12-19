import React from 'react';
import { gql } from '@apollo/client';
import { ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';

import useURLStringUnion from 'hooks/useURLStringUnion';
import { NonEmptyArray } from 'utils/type.utils';
import { SortOption } from 'types/table';
import { entityTabValues, EntityTab } from '../types';
import { getDefaultSortOption } from '../sortUtils';

export type EntityCounts = {
    imageCount: number;
    deploymentCount: number;
    imageCVECount: number;
};

export const entityTypeCountsQuery = gql`
    query getEntityTypeCounts($query: String) {
        imageCount(query: $query)
        deploymentCount(query: $query)
        imageCVECount(query: $query)
    }
`;

type EntityTabToggleGroupProps = {
    className?: string;
    entityTabs?: Readonly<NonEmptyArray<EntityTab>>;
    cveCount?: number;
    imageCount?: number;
    deploymentCount?: number;
    setSortOption: (sortOption: SortOption) => void;
    setPage: (num) => void;
    onChange: (entityTab: EntityTab) => void;
};

function EntityTabToggleGroup({
    className = '',
    entityTabs = entityTabValues,
    cveCount = 0,
    imageCount = 0,
    deploymentCount = 0,
    setSortOption,
    setPage,
    onChange,
}: EntityTabToggleGroupProps) {
    const [activeEntityTabKey, setActiveEntityTabKey] = useURLStringUnion('entityTab', entityTabs);

    function handleEntityTabChange(entityTab: EntityTab) {
        setActiveEntityTabKey(entityTab);
        setSortOption(getDefaultSortOption(entityTab));
        setPage(1);
        onChange(entityTab);
    }

    return (
        <ToggleGroup className={className} aria-label="Entity type toggle items">
            {entityTabs.includes('CVE') ? (
                <ToggleGroupItem
                    text={`${cveCount} CVEs`}
                    buttonId="cves"
                    isSelected={activeEntityTabKey === 'CVE'}
                    onChange={() => handleEntityTabChange('CVE')}
                />
            ) : (
                <></>
            )}
            {entityTabs.includes('Image') ? (
                <ToggleGroupItem
                    text={`${imageCount} Images`}
                    buttonId="images"
                    isSelected={activeEntityTabKey === 'Image'}
                    onChange={() => handleEntityTabChange('Image')}
                />
            ) : (
                <></>
            )}
            {entityTabs.includes('Deployment') ? (
                <ToggleGroupItem
                    text={`${deploymentCount} Deployments`}
                    buttonId="deployments"
                    isSelected={activeEntityTabKey === 'Deployment'}
                    onChange={() => handleEntityTabChange('Deployment')}
                />
            ) : (
                <></>
            )}
        </ToggleGroup>
    );
}

export default EntityTabToggleGroup;
