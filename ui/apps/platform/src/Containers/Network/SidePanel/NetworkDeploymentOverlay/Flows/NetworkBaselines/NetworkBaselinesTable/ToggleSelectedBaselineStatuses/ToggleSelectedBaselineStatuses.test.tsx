import { getSelectedRows } from './ToggleSelectedBaselineStatuses';

import { Row } from '../tableTypes';

describe('ToggleSelectedBaselineStatuses', () => {
    describe('getSelectedRows', () => {
        it('should get the selected rows', () => {
            const selectedFlatRows = [
                {
                    id: 'status:BASELINE>entity:External Entities',
                    groupByID: 'entity',
                    isGrouped: true,
                    groupByVal: 'External Entities',
                    subRows: [
                        {
                            id: '0',
                            original: {
                                peer: {
                                    entity: {
                                        id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                                        name: 'External Entities',
                                        namespace: '-',
                                        type: 'INTERNET',
                                    },
                                    ingress: true,
                                    port: '9443',
                                    protocol: 'L4_PROTOCOL_TCP',
                                    state: 'active',
                                },
                                status: 'BASELINE',
                            },
                            values: { status: 'BASELINE ' },
                        },
                        {
                            id: '1',
                            original: {
                                peer: {
                                    entity: {
                                        id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                                        name: 'External Entities',
                                        namespace: '-',
                                        type: 'INTERNET',
                                    },
                                    ingress: false,
                                    port: '443',
                                    protocol: 'L4_PROTOCOL_TCP',
                                    state: 'active',
                                },
                                status: 'BASELINE',
                            },
                            values: { status: 'BASELINE ' },
                        },
                    ],
                    values: { status: 'BASELINE ' },
                },
                {
                    id: '0',
                    original: {
                        peer: {
                            entity: {
                                id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                                name: 'External Entities',
                                namespace: '-',
                                type: 'INTERNET',
                            },
                            ingress: true,
                            port: '9443',
                            protocol: 'L4_PROTOCOL_TCP',
                            state: 'active',
                        },
                        status: 'BASELINE',
                    },
                    values: { status: 'BASELINE' },
                },
                {
                    id: '1',
                    original: {
                        peer: {
                            entity: {
                                id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                                name: 'External Entities',
                                namespace: '-',
                                type: 'INTERNET',
                            },
                            ingress: false,
                            port: '443',
                            protocol: 'L4_PROTOCOL_TCP',
                            state: 'active',
                        },
                        status: 'BASELINE',
                    },
                    values: { status: 'BASELINE ' },
                },
            ] as Row[];
            expect(getSelectedRows(selectedFlatRows)).toEqual([
                {
                    peer: {
                        entity: {
                            id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                            name: 'External Entities',
                            namespace: '-',
                            type: 'INTERNET',
                        },
                        ingress: true,
                        port: '9443',
                        protocol: 'L4_PROTOCOL_TCP',
                        state: 'active',
                    },
                    status: 'BASELINE',
                },
                {
                    peer: {
                        entity: {
                            id: 'afa12424-bde3-4313-b810-bb463cbe8f90',
                            name: 'External Entities',
                            namespace: '-',
                            type: 'INTERNET',
                        },
                        ingress: false,
                        port: '443',
                        protocol: 'L4_PROTOCOL_TCP',
                        state: 'active',
                    },
                    status: 'BASELINE',
                },
            ]);
        });
    });
});
