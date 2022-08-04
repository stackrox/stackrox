import { Button, Chip, ChipGroup, Flex, FlexItem } from '@patternfly/react-core';
import React, { ReactElement } from 'react';

import { SearchFilter } from 'types/search';

export type VulnerabilityRequestSearchResultsProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
};

function VulnerabilityRequestSearchResults({
    searchFilter,
    setSearchFilter,
}: VulnerabilityRequestSearchResultsProps): ReactElement {
    const attributes = Object.keys(searchFilter);

    function deleteAll() {
        setSearchFilter({});
    }

    function deleteChipGroup(attribute) {
        const modifiedSearchFilter = { ...searchFilter };
        delete modifiedSearchFilter[attribute];
        setSearchFilter(modifiedSearchFilter);
    }

    function deleteChip(attribute, value) {
        const modifiedSearchFilter = { ...searchFilter };
        const attributeValue = searchFilter[attribute];
        if (Array.isArray(attributeValue)) {
            modifiedSearchFilter[attribute] = attributeValue.filter(
                (filterValue) => filterValue === value
            );
        } else {
            delete modifiedSearchFilter[attribute];
        }
        setSearchFilter(modifiedSearchFilter);
    }

    const chipGroups = attributes.map((attribute) => {
        const attributeValue = searchFilter[attribute];
        const values = !Array.isArray(attributeValue) ? [attributeValue] : attributeValue;
        return (
            <ChipGroup
                categoryName={attribute}
                isClosable
                onClick={() => deleteChipGroup(attribute)}
            >
                {values.map((value) => (
                    <Chip key={value} onClick={() => deleteChip(attribute, value)}>
                        {value}
                    </Chip>
                ))}
            </ChipGroup>
        );
    });

    return (
        <Flex>
            {chipGroups.map((chipGroup) => {
                return <FlexItem spacer={{ default: 'spacerMd' }}>{chipGroup}</FlexItem>;
            })}
            <FlexItem>
                <Button type="button" variant="link" isInline onClick={() => deleteAll()}>
                    Clear all filters
                </Button>
            </FlexItem>
        </Flex>
    );
}

export default VulnerabilityRequestSearchResults;
