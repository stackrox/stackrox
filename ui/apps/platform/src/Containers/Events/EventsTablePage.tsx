import React, { ReactElement, useEffect, useState } from 'react';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { Divider, PageSection, Title } from '@patternfly/react-core';
import { fetchEventSource } from '@microsoft/fetch-event-source';
import { EventProto } from '../../types/event.proto';
import { fetchEvents } from '../../services/EventService';
import { getAccessToken } from '../../services/AuthService';

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

    useEffect(() => {
        const fetchData = async () => {
            await fetchEventSource(`/api/event/stream`, {
                method: 'POST',
                headers: {
                    Accept: 'text/event-stream',
                    Authorization: getAccessToken() as string,
                },
                async onopen(res) {
                    if (res.ok && res.status === 200) {
                        // eslint-disable-next-line no-console
                        console.log('Connection made ', res);
                    } else {
                        // eslint-disable-next-line no-console
                        console.log('Some error ', res);
                    }
                },
                onmessage(event) {
                    // eslint-disable-next-line no-console
                    console.log(event.data);
                    const parsedItem = JSON.parse(event.data);
                    // eslint-disable-next-line @typescript-eslint/no-unsafe-return
                    setItems((events) => [...events, parsedItem]);
                },
                onclose() {
                    // eslint-disable-next-line no-console
                    console.log('Connection closed');
                },
                onerror(err) {
                    // eslint-disable-next-line no-console
                    console.log('Some error ', err);
                },
            });
        };
        // eslint-disable-next-line no-void
        void fetchData();
    }, []);

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
