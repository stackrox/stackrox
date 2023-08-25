import React from 'react';
import { Bullseye, Button, Text } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { Tbody, Tr, Td } from '@patternfly/react-table';

import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import useURLSearch from 'hooks/useURLSearch';

export type EmptyTableResultsProps = {
    colSpan: number;
};

function EmptyTableResults({ colSpan }: EmptyTableResultsProps) {
    const { setSearchFilter } = useURLSearch();
    return (
        <Tbody>
            <Tr>
                <Td colSpan={colSpan}>
                    <Bullseye>
                        <EmptyStateTemplate
                            title="No results found"
                            headingLevel="h2"
                            icon={SearchIcon}
                        >
                            <Text>Clear all filters and try again.</Text>
                            <Button variant="link" onClick={() => setSearchFilter({})}>
                                Clear filters
                            </Button>
                        </EmptyStateTemplate>
                    </Bullseye>
                </Td>
            </Tr>
        </Tbody>
    );
}

export default EmptyTableResults;
