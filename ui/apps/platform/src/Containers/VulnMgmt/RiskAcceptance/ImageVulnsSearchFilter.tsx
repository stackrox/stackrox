import React, { ReactElement, useEffect, useState } from 'react';
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

export type ImageVulnsSearchFilterProps = {
    searchFilter: SearchFilter;
    setSearchFilter: React.Dispatch<React.SetStateAction<SearchFilter>>;
};

function ImageVulnsSearchFilter({
    searchFilter,
    setSearchFilter,
}: ImageVulnsSearchFilterProps): ReactElement {
    const [selectedAttribute, setSelectedAttribute] = useState<string>('');
    const [inputValue, setInputValue] = useState<string>('');

    useEffect(() => {
        if (!searchFilter.CVE) {
            setInputValue('');
        }
    }, [searchFilter]);

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
                    <SelectOption value="CVE">CVE</SelectOption>
                    <SelectOption value="Fixable">Fixable</SelectOption>
                    <SelectOption value="Severity">Severity</SelectOption>
                </SelectSingle>
            </FlexItem>
            <FlexItem spacer={{ default: 'spacerNone' }}>
                {selectedAttribute === 'CVE' && (
                    <InputGroup>
                        <TextInput
                            name="cveSearchInput"
                            id="cveSearchInput"
                            type="search"
                            aria-label="CVE search input"
                            onChange={handleInputChange}
                            value={inputValue}
                        />
                        <Button
                            variant={ButtonVariant.control}
                            aria-label="search button for CVE search input"
                            onClick={() => handleSearchChange(inputValue)}
                        >
                            <SearchIcon />
                        </Button>
                    </InputGroup>
                )}
                {selectedAttribute === 'Fixable' && (
                    <CheckboxSelect
                        ariaLabel="fixable checkbox select"
                        selections={(searchFilter.Fixable || []) as string[]}
                        onChange={handleSearchChange}
                    >
                        <SelectOption value="true">True</SelectOption>
                        <SelectOption value="false">False</SelectOption>
                    </CheckboxSelect>
                )}
                {selectedAttribute === 'Severity' && (
                    <CheckboxSelect
                        ariaLabel="severity checkbox select"
                        selections={(searchFilter.Severity || []) as string[]}
                        onChange={handleSearchChange}
                    >
                        <SelectOption value="LOW_VULNERABILITY_SEVERITY">Low</SelectOption>
                        <SelectOption value="MODERATE_VULNERABILITY_SEVERITY">
                            Moderate
                        </SelectOption>
                        <SelectOption value="IMPORTANT_VULNERABILITY_SEVERITY">
                            Important
                        </SelectOption>
                        <SelectOption value="CRITICAL_VULNERABILITY_SEVERITY">
                            Critical
                        </SelectOption>
                    </CheckboxSelect>
                )}
            </FlexItem>
        </Flex>
    );
}

export default ImageVulnsSearchFilter;
