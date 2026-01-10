import type { ReactElement } from 'react';
import { Button, Flex, FlexItem, Label, LabelGroup } from '@patternfly/react-core';
import { Globe } from 'react-feather'; // eslint-disable-line limited/no-feather-icons

import type { SearchFilter } from 'types/search';

import type { CompoundSearchFilterAttribute, CompoundSearchFilterConfig } from '../types';
import { getCompoundSearchFilterLabelDescriptionOrNull, updateSearchFilter } from '../utils/utils';
import type { CompoundSearchFilterLabelDescription, IsGlobalPredicate } from '../utils/utils';

import './SearchFilterChips.css';

const isGlobalPredicateFalse: IsGlobalPredicate = () => false;

const iconGlobe = <Globe height="15px" />;

export type CompoundSearchFilterLabelsProps = {
    attributesSeparateFromConfig: CompoundSearchFilterAttribute[];
    config: CompoundSearchFilterConfig;
    isGlobalPredicate?: IsGlobalPredicate; // for certain values in AdvancedFilterToolbar.tsx file
    onFilterChange?: (searchFilter: SearchFilter) => void; // omit for view-based report details
    searchFilter: SearchFilter;
};

function CompoundSearchFilterLabels({
    attributesSeparateFromConfig,
    config,
    isGlobalPredicate = isGlobalPredicateFalse,
    onFilterChange,
    searchFilter,
}: CompoundSearchFilterLabelsProps): ReactElement {
    const attributesFromConfig = config.flatMap(({ attributes }) => attributes);
    const attributes = [...attributesFromConfig, ...attributesSeparateFromConfig];
    const labelGroupDescriptions: CompoundSearchFilterLabelDescription[] = [];
    attributes.forEach((attribute) => {
        const labelDescriptionOrNull = getCompoundSearchFilterLabelDescriptionOrNull(
            attribute,
            searchFilter,
            isGlobalPredicate
        );
        if (labelDescriptionOrNull !== null) {
            // Attribute has one or more values in the search filter.
            labelGroupDescriptions.push(labelDescriptionOrNull);
        }
    });

    return (
        <Flex className="search-filter-chips" spaceItems={{ default: 'spaceItemsXs' }}>
            {labelGroupDescriptions.map(({ group, items }) => {
                return (
                    <FlexItem key={group.label}>
                        <LabelGroup
                            categoryName={group.label}
                            isClosable={Boolean(onFilterChange)}
                            onClick={
                                onFilterChange &&
                                (() =>
                                    onFilterChange(updateSearchFilter(searchFilter, group.payload)))
                            }
                        >
                            {items.map((item) => (
                                <Label
                                    key={item.label}
                                    variant="outline"
                                    icon={item.isGlobal ? iconGlobe : undefined}
                                    closeBtnAriaLabel="Remove filter"
                                    onClose={
                                        onFilterChange &&
                                        (() =>
                                            onFilterChange(
                                                updateSearchFilter(searchFilter, item.payload)
                                            ))
                                    }
                                >
                                    {item.label}
                                </Label>
                            ))}
                        </LabelGroup>
                    </FlexItem>
                );
            })}
            {labelGroupDescriptions.length !== 0 && onFilterChange && (
                <Button variant="link" onClick={() => onFilterChange({})}>
                    Clear filters
                </Button>
            )}
        </Flex>
    );
}

export default CompoundSearchFilterLabels;
