import React from 'react';
import {
    CodeBlock,
    CodeBlockCode,
    Divider,
    Flex,
    FlexItem,
    PageSection,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

import useURLSearch from 'hooks/useURLSearch';
import { compoundSearchFilter } from 'Components/CompoundSearchFilter/types';

import PageTitle from 'Components/PageTitle';
import CompoundSearchFilter from 'Components/CompoundSearchFilter/components/CompoundSearchFilter';
import SearchFilterChips from 'Components/CompoundSearchFilter/components/SearchFilterChips';

function DemoPage() {
    const { searchFilter, setSearchFilter } = useURLSearch();

    return (
        <>
            <PageTitle title="Demo - Advanced Filters" />
            <PageSection variant="light">
                <Flex>
                    <Flex direction={{ default: 'column' }} flex={{ default: 'flex_1' }}>
                        <Title headingLevel="h1">Demo - Advanced Filters</Title>
                        <FlexItem>
                            This section will demo the capabilities of advanced filters. NOT A REAL
                            PAGE
                        </FlexItem>
                    </Flex>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <PageSection variant="light">
                    <Toolbar>
                        <ToolbarContent>
                            <ToolbarItem widths={{ default: '100%' }}>
                                <CompoundSearchFilter
                                    config={compoundSearchFilter}
                                    onSearch={(searchKey, searchValue) => {
                                        setSearchFilter({
                                            ...searchFilter,
                                            [searchKey]: searchValue,
                                        });
                                    }}
                                />
                            </ToolbarItem>
                            <ToolbarItem>
                                <SearchFilterChips
                                    searchFilter={searchFilter}
                                    setSearchFilter={setSearchFilter}
                                />
                            </ToolbarItem>
                        </ToolbarContent>
                    </Toolbar>
                </PageSection>
            </PageSection>
        </>
    );
}

export default DemoPage;
