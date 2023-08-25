import React, { ReactElement, useState } from 'react';
import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    InputGroup,
    SelectOption,
    TextInput,
} from '@patternfly/react-core';
import { FilterIcon, SearchIcon } from '@patternfly/react-icons';

import { SearchFilter } from 'types/search';
import CheckboxSelect from 'Components/PatternFly/CheckboxSelect';
import SelectSingle from 'Components/SelectSingle';

export type ApprovedDeferralsSearchFilterProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
};

function ApprovedDeferralsSearchFilter({
    searchFilter,
    setSearchFilter,
}: ApprovedDeferralsSearchFilterProps): ReactElement {
    const [selectedAttribute, setSelectedAttribute] = useState<string>('');
    const [inputValue, setInputValue] = useState<string>('');

    function handleSelectedAttribute(_, value: string) {
        setSelectedAttribute(value);
    }

    function handleSearchChange(value) {
        const modifiedSearchObject = { ...searchFilter };
        if (value === '' || (Array.isArray(value) && value.length === 0)) {
            delete modifiedSearchObject[selectedAttribute];
        } else {
            modifiedSearchObject[selectedAttribute] = value;
        }
        setSearchFilter(modifiedSearchObject);
    }

    function handleInputChange(value) {
        setInputValue(value);
    }

    // @TODO: We want to change these to sentence case and change the data structure for search filter
    // accordingly
    return (
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            <FlexItem spacer={{ default: 'spacerNone' }}>
                <SelectSingle
                    toggleIcon={<FilterIcon />}
                    id="search-filter-attributes-select"
                    value={selectedAttribute}
                    handleSelect={handleSelectedAttribute}
                    placeholderText="Select a filter..."
                >
                    <SelectOption value="Request ID">Request ID</SelectOption>
                    <SelectOption value="Request Status">Request Status</SelectOption>
                </SelectSingle>
            </FlexItem>
            <FlexItem spacer={{ default: 'spacerNone' }}>
                {selectedAttribute === 'Request ID' && (
                    <InputGroup>
                        <TextInput
                            name="requestIDSearchInput"
                            id="requestIDSearchInput"
                            type="search"
                            aria-label="request id search input"
                            onChange={handleInputChange}
                            value={inputValue}
                        />
                        <Button
                            variant={ButtonVariant.control}
                            aria-label="search button for search input"
                            onClick={() => handleSearchChange(inputValue)}
                        >
                            <SearchIcon />
                        </Button>
                    </InputGroup>
                )}
                {selectedAttribute === 'Request Status' && (
                    <CheckboxSelect
                        ariaLabel="request status checkbox select"
                        selections={(searchFilter['Request Status'] || []) as string[]}
                        onChange={handleSearchChange}
                    >
                        <SelectOption value="APPROVED">Approved</SelectOption>
                        <SelectOption value="APPROVED_PENDING_UPDATE">
                            Approved - Pending Update
                        </SelectOption>
                    </CheckboxSelect>
                )}
            </FlexItem>
        </Flex>
    );
}

export default ApprovedDeferralsSearchFilter;
