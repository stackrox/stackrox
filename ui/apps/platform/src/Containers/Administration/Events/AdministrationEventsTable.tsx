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

import IconText from 'Components/PatternFly/IconText/IconText';
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
                        <Th>Domain</Th>
                        <Th modifier="nowrap">Resource type</Th>
                        <Th>Level</Th>
                        <Th>Event last occurred at</Th>
                        <Th className="pf-u-text-align-right">Count</Th>
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
                        resource,
                    } = event;
                    const { type: resourceType } = resource;

                    return (
                        <Tbody
                            key={id}
                            isExpanded
                            style={{
                                borderBottom: '1px solid var(--pf-c-table--BorderColor)',
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
                                <Td dataLabel="Event last occurred at" modifier="nowrap">
                                    {lastOccurredAt}
                                </Td>
                                <Td dataLabel="Count" className="pf-u-text-align-right">
                                    {numOccurrences}
                                </Td>
                            </Tr>
                            <Tr>
                                <Td colSpan={5}>
                                    <ExpandableRowContent>
                                        <Flex direction={{ default: 'column' }}>
                                            {hint && (
                                                <div>
                                                    {hint.split('\n').map((line) => (
                                                        <p key={line}>{line}</p>
                                                    ))}
                                                </div>
                                            )}
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
