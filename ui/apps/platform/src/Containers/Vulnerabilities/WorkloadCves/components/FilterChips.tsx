import React from 'react';
import { Globe } from 'react-feather';
import { Flex, FlexItem, ChipGroup, Chip, Button } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';
import { VulnerabilitySeverityLabel, FixableStatus, DefaultFilters } from '../types';

import './FilterChips.css';

type FilterChipProps = {
    isGlobal?: boolean;
    name: string;
};

function FilterChip({ isGlobal, name }: FilterChipProps) {
    if (isGlobal) {
        return (
            <Flex alignItems={{ default: 'alignItemsCenter' }} flexWrap={{ default: 'nowrap' }}>
                <Globe height="15px" />
                {name}
            </Flex>
        );
    }
    return <Flex>{name}</Flex>;
}

type FilterChipsProps = {
    defaultFilters: DefaultFilters;
    searchFilter: SearchFilter;
    onDeleteGroup: (category) => void;
    onDelete: (category, chip) => void;
    onDeleteAll: () => void;
};

function FilterChips({
    defaultFilters,
    searchFilter,
    onDeleteGroup,
    onDelete,
    onDeleteAll,
}: FilterChipsProps) {
    const deployments = (searchFilter.DEPLOYMENT as string[]) || [];
    const cves = (searchFilter.CVE as string[]) || [];
    const images = (searchFilter.IMAGE as string[]) || [];
    const namespaces = (searchFilter.NAMESPACE as string[]) || [];
    const clusters = (searchFilter.CLUSTER as string[]) || [];
    const severities = (searchFilter.Severity as VulnerabilitySeverityLabel[]) || [];
    const fixables = (searchFilter.Fixable as FixableStatus[]) || [];

    return (
        <Flex spaceItems={{ default: 'spaceItemsXs' }}>
            {deployments.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup
                        categoryName="Deployment"
                        isClosable
                        onClick={() => onDeleteGroup('DEPLOYMENT')}
                    >
                        {deployments.map((deployment) => (
                            <Chip
                                key={deployment}
                                onClick={() => onDelete('DEPLOYMENT', deployment)}
                            >
                                {deployment}
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            {cves.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup categoryName="CVE" isClosable onClick={() => onDeleteGroup('CVE')}>
                        {cves.map((cve) => (
                            <Chip key={cve} onClick={() => onDelete('CVE', cve)}>
                                {cve}
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            {images.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup
                        categoryName="Image"
                        isClosable
                        onClick={() => onDeleteGroup('IMAGE')}
                    >
                        {images.map((image) => (
                            <Chip key={image} onClick={() => onDelete('IMAGE', image)}>
                                {image}
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            {namespaces.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup
                        categoryName="Namespace"
                        isClosable
                        onClick={() => onDeleteGroup('NAMESPACE')}
                    >
                        {namespaces.map((namespace) => (
                            <Chip key={namespace} onClick={() => onDelete('NAMESPACE', namespace)}>
                                {namespace}
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            {clusters.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup
                        categoryName="Cluster"
                        isClosable
                        onClick={() => onDeleteGroup('CLUSTER')}
                    >
                        {clusters.map((cluster) => (
                            <Chip key={cluster} onClick={() => onDelete('CLUSTER', cluster)}>
                                {cluster}
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            {severities.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup
                        categoryName="Severity"
                        isClosable
                        onClick={() => onDeleteGroup('Severity')}
                    >
                        {severities.map((severity) => (
                            <Chip key={severity} onClick={() => onDelete('Severity', severity)}>
                                <FilterChip
                                    isGlobal={defaultFilters.Severity?.includes(severity)}
                                    name={severity}
                                />
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            {fixables.length > 0 && (
                <FlexItem className="pf-u-pt-xs">
                    <ChipGroup
                        categoryName="Fixable"
                        isClosable
                        onClick={() => onDeleteGroup('Fixable')}
                    >
                        {fixables.map((fixableStatus) => (
                            <Chip
                                key={fixableStatus}
                                onClick={() => onDelete('Fixable', fixableStatus)}
                            >
                                <FilterChip
                                    isGlobal={defaultFilters.Fixable?.includes(fixableStatus)}
                                    name={fixableStatus}
                                />
                            </Chip>
                        ))}
                    </ChipGroup>
                </FlexItem>
            )}
            <Button variant="link" onClick={onDeleteAll}>
                Clear filters
            </Button>
        </Flex>
    );
}

export default FilterChips;
