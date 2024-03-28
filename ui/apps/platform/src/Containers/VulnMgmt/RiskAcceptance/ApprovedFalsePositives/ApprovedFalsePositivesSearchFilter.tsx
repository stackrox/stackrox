import React, { ReactElement, useState } from 'react';
import {
    Button,
    ButtonVariant,
    InputGroup,
    TextInput,
    InputGroupItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import { SearchFilter } from 'types/search';

export type ApprovedFalsePositivesSearchFilterProps = {
    searchFilter: SearchFilter;
    setSearchFilter: (newFilter: SearchFilter) => void;
};

function ApprovedFalsePositivesSearchFilter({
    searchFilter,
    setSearchFilter,
}: ApprovedFalsePositivesSearchFilterProps): ReactElement {
    const [inputValue, setInputValue] = useState<string>('');

    const selectedAttribute = 'Request ID';

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

    return (
        <InputGroup>
            <InputGroupItem isFill>
                <TextInput
                    name="requestIDSearchInput"
                    id="requestIDSearchInput"
                    type="search"
                    aria-label="request id search input"
                    placeholder="Filter by request ID"
                    onChange={(_event, value) => handleInputChange(value)}
                    value={inputValue}
                />
            </InputGroupItem>
            <InputGroupItem>
                <Button
                    variant={ButtonVariant.control}
                    aria-label="search button for search input"
                    onClick={() => handleSearchChange(inputValue)}
                >
                    <SearchIcon />
                </Button>
            </InputGroupItem>
        </InputGroup>
    );
}

export default ApprovedFalsePositivesSearchFilter;
