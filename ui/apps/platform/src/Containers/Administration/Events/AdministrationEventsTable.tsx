import type { ReactElement } from 'react';
import { Link } from 'react-router-dom-v5-compat';
import { ExpandableRowContent, Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import IconText from 'Components/PatternFly/IconText/IconText';
import type { UseURLSortResult } from 'hooks/useURLSort';
import {
    hasAdministrationEventsFilter,
    lastOccurredAtField,
    numOccurrencesField,
} from 'services/AdministrationEventsService';
import type { AdministrationEvent } from 'services/AdministrationEventsService';
import type { SearchFilter } from 'types/search';

import { getLevelIcon, getLevelText } from './AdministrationEvent';
import AdministrationEventHintMessage from './AdministrationEventHintMessage';

import AdministrationEventsEmptyState from './AdministrationEventsEmptyState';

const colSpan = 5;

export type AdministrationEventsTableProps = {
    events: AdministrationEvent[];
    getSortParams: UseURLSortResult['getSortParams'];
    searchFilter: SearchFilter;
};

function AdministrationEventsTable({
    events,
    getSortParams,
    searchFilter,
}: AdministrationEventsTableProps): ReactElement {
    return (
        <Table variant="compact" borders={false}>
            <Thead>
                <Tr>
                    <Th>Domain</Th>
                    <Th modifier="nowrap">Resource type</Th>
                    <Th>Level</Th>
                    <Th sort={getSortParams(lastOccurredAtField)}>Last occurred</Th>
                    <Th sort={getSortParams(numOccurrencesField)}>Count</Th>
                </Tr>
            </Thead>
            {events.length === 0 ? (
                <AdministrationEventsEmptyState
                    colSpan={colSpan}
                    hasFilter={hasAdministrationEventsFilter(searchFilter)}
                />
            ) : (
                events.map((event) => {
                    const { domain, id, lastOccurredAt, level, numOccurrences, resource } = event;
                    const { type: resourceType } = resource;

                    return (
                        <Tbody
                            key={id}
                            isExpanded
                            style={{
                                borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                            }}
                        >
                            <Tr>
                                <Td dataLabel="Domain" modifier="nowrap">
                                    <Link to={`/main/administration-events/${id}`}>{domain}</Link>
                                </Td>
                                <Td dataLabel="Resource type" modifier="nowrap">
                                    {resourceType}
                                </Td>
                                <Td dataLabel="Level">
                                    <IconText
                                        icon={getLevelIcon(level)}
                                        text={getLevelText(level)}
                                    />
                                </Td>
                                <Td dataLabel="Last occurred" modifier="nowrap">
                                    {lastOccurredAt}
                                </Td>
                                <Td dataLabel="Count">{numOccurrences}</Td>
                            </Tr>
                            <Tr>
                                <Td colSpan={colSpan}>
                                    <ExpandableRowContent>
                                        <AdministrationEventHintMessage event={event} />
                                    </ExpandableRowContent>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                })
            )}
        </Table>
    );
}

export default AdministrationEventsTable;
