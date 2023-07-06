import React from 'react';
import { Divider, PageSection, Title } from '@patternfly/react-core';

import PageTitle from 'Components/PageTitle';
import { ProcessListeningOnPort } from 'services/ProcessListeningOnPortsService';
import ListeningEndpointsTable from './ListeningEndpointsTable';

const listeningEndpointsSampleData: ProcessListeningOnPort[] = [
    {
        endpoint: {
            port: 9090,
            protocol: 'L4_PROTOCOL_TCP',
        },
        deploymentId: 'c83ca25c-1097-4dd0-bca1-1df5c496ac5e',
        containerName: 'central',
        podId: 'central-5d88f6fcb5-zvgw5',
        podUid: '79d49039-a2b3-5d05-b929-e74bc82da71c',
        signal: {
            id: '706f77e8-167d-11ee-a384-fe93f92a2b33',
            containerId: '3ccef27a1cf5',
            time: '2023-06-29T13:00:52.138198746Z',
            name: 'central',
            args: '',
            execFilePath: '/stackrox/central',
            pid: 12530,
            uid: 4000,
            gid: 0,
            lineage: [],
            scraped: true,
            lineageInfo: [],
        },
        clusterId: '9b217bfd-1c3f-4836-9d57-c0b6d410ccda',
        namespace: 'stackrox',
        containerStartTime: '2023-06-29T13:00:52Z',
        imageId: 'sha256:a245029c808486cd18e1ade1a4c1d2db9210cfa87ffa894f5837df37e4bccebd',
    },
    {
        endpoint: {
            port: 8443,
            protocol: 'L4_PROTOCOL_TCP',
        },
        deploymentId: 'c83ca25c-1097-4dd0-bca1-1df5c496ac5e',
        containerName: 'central',
        podId: 'central-5d88f6fcb5-zvgw5',
        podUid: '79d49039-a2b3-5d05-b929-e74bc82da71c',
        signal: {
            id: '706f77e8-167d-11ee-a384-fe93f92a2b33',
            containerId: '3ccef27a1cf5',
            time: '2023-06-29T13:00:52.138198746Z',
            name: 'central',
            args: '',
            execFilePath: '/stackrox/central',
            pid: 12530,
            uid: 4000,
            gid: 0,
            lineage: [],
            scraped: true,
            lineageInfo: [],
        },
        clusterId: '9b217bfd-1c3f-4836-9d57-c0b6d410ccda',
        namespace: 'stackrox',
        containerStartTime: '2023-06-29T13:00:52Z',
        imageId: 'sha256:a245029c808486cd18e1ade1a4c1d2db9210cfa87ffa894f5837df37e4bccebd',
    },
];

function ListeningEndpointsPage() {
    return (
        <>
            <PageTitle title="Listening Endpoints" />
            <PageSection variant="light">
                <Title headingLevel="h1">Listening endpoints</Title>
            </PageSection>
            <Divider component="div" />
            <PageSection isFilled>
                <ListeningEndpointsTable listeningEndpoints={listeningEndpointsSampleData} />
            </PageSection>
        </>
    );
}

export default ListeningEndpointsPage;
