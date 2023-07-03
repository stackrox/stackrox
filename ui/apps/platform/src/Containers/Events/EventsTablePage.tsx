import React, { ReactElement, useEffect, useState } from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Divider, PageSection, Title } from '@patternfly/react-core';
import { EventProto } from '../../types/event.proto';
import { fetchEvents } from '../../services/EventService';

function EventsTablePage(): ReactElement {
    const [items, setItems] = useState<EventProto[]>([]);
    useEffect(() => {
        fetchEvents()
            .then((itemsFetched) => {
                setItems(itemsFetched.response.events);
            })
            .catch(() => {
                setItems([]);
            });
    });

    return (
        <>
            <PageSection variant="light" id="events-table">
                <Title headingLevel="h1">Events</Title>
                <Divider className="pf-u-py-md" />
            </PageSection>
            <PageSection variant="light">
                <TableComposable variant="compact">
                    <Thead>
                        <Tr>
                            <Th width={40}>ID</Th>
                            <Th width={80}>Message</Th>
                        </Tr>
                    </Thead>
                    <Tbody data-testid="events">
                        {items.map(({ id, msg }) => (
                            <Tr key={id}>
                                <Td dataLabel="ID" modifier="breakWord" data-testid="event-id">
                                    {id}
                                </Td>
                                <Td
                                    dataLabel="Message"
                                    modifier="breakWord"
                                    data-testid="event-message"
                                >
                                    {msg}
                                </Td>
                            </Tr>
                        ))}
                    </Tbody>
                </TableComposable>
            </PageSection>
        </>
    );
}

export default EventsTablePage;
