import React, { ReactElement, useState } from 'react';
import {
    Button,
    ButtonVariant,
    Flex,
    FlexItem,
    InputGroup,
    TextInput,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';

import { SearchFilter } from 'types/search';

export type ReportsSearchFilterProps = {
    searchFilter: SearchFilter;
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
};

function ReportsSearchFilter({
    searchFilter,
    setSearchFilter,
}: ReportsSearchFilterProps): ReactElement {
    const [selectedAttribute] = useState<string>('Report Name');
    const [inputValue, setInputValue] = useState<string>('');

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
        <Flex alignItems={{ default: 'alignItemsCenter' }}>
            <FlexItem spacer={{ default: 'spacerNone' }}>
                {selectedAttribute === 'Report Name' && (
                    <InputGroup>
                        <TextInput
                            name="reportNameSearchInput"
                            id="reportNameSearchInput"
                            type="search"
                            aria-label="Report name search input"
                            onChange={handleInputChange}
                            value={inputValue}
                        />
                        <Button
                            variant={ButtonVariant.control}
                            aria-label="Perform search"
                            onClick={() => handleSearchChange(inputValue)}
                        >
                            <SearchIcon />
                        </Button>
                    </InputGroup>
                )}
            </FlexItem>
        </Flex>
    );
}

export default ReportsSearchFilter;
