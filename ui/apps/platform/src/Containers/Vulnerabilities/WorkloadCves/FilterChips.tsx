/* eslint-disable react/no-children-prop */
import React from 'react';
import { Globe } from 'react-feather';
import { Flex, ToolbarChip, ToolbarFilter } from '@patternfly/react-core';

import { SearchFilter } from 'types/search';
import { VulnerabilitySeverityLabel, FixableStatus, DefaultFilters } from './types';

import './FilterChips.css';

type FilterChipsProps = {
    defaultFilters: DefaultFilters;
    searchFilter: SearchFilter;
    onDeleteGroup: (category) => void;
    onDelete: (category, chip) => void;
};

function FilterChips({ defaultFilters, searchFilter, onDeleteGroup, onDelete }: FilterChipsProps) {
    const severityFilterChips: ToolbarChip[] = [];
    const fixableFilterChips: ToolbarChip[] = [];
    const severitySearchFilter = searchFilter.Severity as VulnerabilitySeverityLabel[];
    severitySearchFilter?.forEach((sev) => {
        if (defaultFilters.Severity?.includes(sev)) {
            severityFilterChips.push({
                key: sev,
                node: (
                    <Flex alignItems={{ default: 'alignItemsCenter' }}>
                        <Globe height="15px" />
                        {sev}
                    </Flex>
                ),
            });
        } else {
            severityFilterChips.push({
                key: sev,
                node: <Flex>{sev}</Flex>,
            });
        }
    });

    const fixableSearchFilter = searchFilter.Fixable as FixableStatus[];
    fixableSearchFilter?.forEach((status) => {
        if (defaultFilters.Fixable?.includes(status)) {
            fixableFilterChips.push({
                key: status,
                node: (
                    <Flex
                        alignItems={{ default: 'alignItemsCenter' }}
                        flexWrap={{ default: 'nowrap' }}
                    >
                        <Globe height="15px" />
                        {status}
                    </Flex>
                ),
            });
        } else {
            fixableFilterChips.push({
                key: status,
                node: <Flex>{status}</Flex>,
            });
        }
    });

    return (
        <>
            {/* adding children as undefined here because we want to show the filter chips even
            when the resource is set to something else in the dropdown
            (children are required for the ToolbarFilter component even though functionally 
            it seems to work fine) */}
            <ToolbarFilter
                chips={searchFilter.DEPLOYMENT ? (searchFilter.DEPLOYMENT as string[]) : []}
                deleteChip={(_, chip) => onDelete('DEPLOYMENT', chip as string)}
                deleteChipGroup={() => onDeleteGroup('DEPLOYMENT')}
                categoryName="Deployment"
                children={undefined}
            />
            <ToolbarFilter
                chips={searchFilter.IMAGE_CVE ? (searchFilter.IMAGE_CVE as string[]) : []}
                deleteChip={(_, chip) => onDelete('IMAGE_CVE', chip as string)}
                deleteChipGroup={() => onDeleteGroup('IMAGE_CVE')}
                categoryName="CVE"
                children={undefined}
            />
            <ToolbarFilter
                chips={searchFilter.IMAGE ? (searchFilter.IMAGE as string[]) : []}
                deleteChip={(_, chip) => onDelete('IMAGE', chip as string)}
                deleteChipGroup={() => onDeleteGroup('IMAGE')}
                categoryName="Image"
                children={undefined}
            />
            <ToolbarFilter
                chips={searchFilter.NAMESPACE ? (searchFilter.NAMESPACE as string[]) : []}
                deleteChip={(_, chip) => onDelete('NAMESPACE', chip as string)}
                deleteChipGroup={() => onDeleteGroup('NAMESPACE')}
                categoryName="Namespace"
                children={undefined}
            />
            <ToolbarFilter
                chips={searchFilter.CLUSTER ? (searchFilter.CLUSTER as string[]) : []}
                deleteChip={(_, chip) => onDelete('CLUSTER', chip as string)}
                deleteChipGroup={() => onDeleteGroup('CLUSTER')}
                categoryName="Cluster"
                children={undefined}
            />
            <ToolbarFilter
                chips={severityFilterChips}
                deleteChip={(_, chip) => onDelete('Severity', chip as ToolbarChip)}
                deleteChipGroup={() => onDeleteGroup('Severity')}
                categoryName="Severity"
                children={undefined}
            />
            <ToolbarFilter
                chips={fixableFilterChips}
                deleteChip={(_, chip) => onDelete('Fixable', chip as ToolbarChip)}
                deleteChipGroup={() => onDeleteGroup('Fixable')}
                categoryName="Fixable"
                children={undefined}
            />
        </>
    );
}

export default FilterChips;
