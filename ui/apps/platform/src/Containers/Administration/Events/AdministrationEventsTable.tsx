import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { CodeBlock, Flex } from '@patternfly/react-core';
import {
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import { AdministrationEvent } from 'services/AdministrationEventsService';

import { getLevelIcon, getLevelText } from './AdministrationEvent';

import './AdministrationEventsTable.css';

export type AdministrationEventsTableProps = {
    events: AdministrationEvent[];
};

function AdministrationEventsTable({ events }: AdministrationEventsTableProps): ReactElement {
    return (
        <>
            <TableComposable variant="compact" borders={false} id="AdministrationEventsTable">
                <Thead>
                    <Tr>
                        <Td />
                        <Th>Level</Th>
                        <Th>Domain</Th>
                        <Th>Resource type</Th>
                        <Th>Event last occurred at</Th>
                        <Th className="pf-u-text-align-right">Occurrences</Th>
                    </Tr>
                </Thead>
                {events.map((event) => {
                    const {
                        domain,
                        hint,
                        id,
                        lastOccurredAt,
                        level,
                        message,
                        numOccurrences,
                        resourceType,
                    } = event;

                    return (
                        <Tbody
                            key={id}
                            isExpanded
                            style={{
                                borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                            }}
                        >
                            <Tr>
                                <Td dataLabel="Level icon">{getLevelIcon(level)}</Td>
                                <Td dataLabel="Level">
                                    <Link to={`/main/administration-events/${id}`}>
                                        {getLevelText(level)}
                                    </Link>
                                </Td>
                                <Td dataLabel="Domain">{domain}</Td>
                                <Td dataLabel="Resource type">{resourceType}</Td>
                                <Td dataLabel="Event last occurred at" modifier="nowrap">
                                    {lastOccurredAt}
                                </Td>
                                <Td dataLabel="Occurrences" className="pf-u-text-align-right">
                                    {numOccurrences}
                                </Td>
                            </Tr>
                            <Tr>
                                <Td />
                                <Td colSpan={6}>
                                    <ExpandableRowContent>
                                        <Flex direction={{ default: 'column' }}>
                                            {hint && <p>{hint}</p>}
                                            <CodeBlock>{message}</CodeBlock>
                                        </Flex>
                                    </ExpandableRowContent>
                                </Td>
                            </Tr>
                        </Tbody>
                    );
                })}
            </TableComposable>
        </>
    );
}

export default AdministrationEventsTable;
