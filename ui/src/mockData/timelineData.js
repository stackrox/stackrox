export const overviewData = {
    deployment: {
        numPolicyViolations: 3,
        numProcessActivities: 5,
        numRestarts: 2,
        numFailures: 2,
        numTotalPods: 3
    }
};
export const podsData = {
    ...overviewData,
    pods: [
        {
            id: 'p-1',
            name: 'hello-world',
            startTime: '2019-12-09T05:51:52Z',
            inactive: false,
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-12-09T06:51:52Z',
                    edges: [],
                    type: 'PolicyViolation'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-12-09T07:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                },
                {
                    processId: 'process-3',
                    timestamp: '2019-12-09T08:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-12-09T09:51:52Z',
                    edges: [],
                    type: 'Restart'
                }
            ]
        },
        {
            id: 'p-2',
            name: 'dr-seuss',
            startTime: '2019-10-09T05:51:52Z',
            inactive: false,
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-10-09T06:51:52Z',
                    edges: [],
                    type: 'PolicyViolation'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-10-09T09:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                },
                {
                    processId: 'process-3',
                    timestamp: '2019-10-09T09:51:52Z',
                    edges: [],
                    type: 'PolicyViolation'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-10-09T10:51:52Z',
                    edges: [],
                    type: 'Restart'
                }
            ]
        },
        {
            id: 'p-3',
            name: 'two-peas-in-a-pod',
            startTime: '2019-11-09T05:51:52Z',
            inactive: false,
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-11-09T07:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-11-09T09:51:52Z',
                    edges: [],
                    type: 'Failure'
                },
                {
                    processId: 'process-3',
                    timestamp: '2019-11-09T07:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-11-09T11:51:52Z',
                    edges: [],
                    type: 'Failure'
                }
            ]
        }
    ]
};

export const containersData = {
    containers: [
        {
            id: 'c-1',
            name: 'hello-1',
            startTime: '2019-12-09T05:51:52Z',
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-12-09T06:51:52Z',
                    edges: [],
                    type: 'PolicyViolation'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-12-09T07:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                }
            ]
        },
        {
            id: 'c-2',
            name: 'hello-2',
            startTime: '2019-12-09T05:51:52Z',
            events: [
                {
                    processId: 'process-3',
                    timestamp: '2019-12-09T08:51:52Z',
                    edges: [],
                    type: 'ProcessActivity'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-12-09T09:51:52Z',
                    edges: [],
                    type: 'Restart'
                }
            ]
        }
    ]
};
