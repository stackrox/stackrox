import { Bullseye, Button, Content } from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { Tbody, Td, Tr } from '@patternfly/react-table';

import EmptyStateTemplate from 'Components/EmptyStateTemplate';
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
                            <Content component="p">Clear all filters and try again.</Content>
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
