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
            numContainers: 3,
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-12-09T06:51:52Z',
                    edges: [],
                    type: 'POLICY_VIOLATION'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-12-09T07:51:52Z',
                    edges: [],
                    type: 'PROCESS_ACTIVITY'
                },
                {
                    processId: 'process-3',
                    timestamp: '2019-12-09T08:51:52Z',
                    edges: [],
                    type: 'PROCESS_ACTIVITY'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-12-09T09:51:52Z',
                    edges: [],
                    type: 'RESTART'
                }
            ]
        },
        {
            id: 'p-2',
            name: 'dr-seuss',
            startTime: '2019-10-09T05:51:52Z',
            inactive: false,
            numContainers: 0,
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-10-09T06:51:52Z',
                    edges: [],
                    type: 'POLICY_VIOLATION'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-10-09T09:51:52Z',
                    edges: [],
                    type: 'PROCESS_ACTIVITY'
                },
                {
                    processId: 'process-3',
                    timestamp: '2019-10-09T09:51:52Z',
                    edges: [],
                    type: 'POLICY_VIOLATION'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-10-09T10:51:52Z',
                    edges: [],
                    type: 'RESTART'
                }
            ]
        },
        {
            id: 'p-3',
            name: 'two-peas-in-a-pod',
            startTime: '2019-11-09T05:51:52Z',
            inactive: false,
            numContainers: 1,
            events: [
                {
                    processId: 'process-1',
                    timestamp: '2019-11-09T07:51:52Z',
                    edges: [],
                    type: 'PROCESS_ACTIVITY'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-11-09T09:51:52Z',
                    edges: [],
                    type: 'FAILURE'
                },
                {
                    processId: 'process-3',
                    timestamp: '2019-11-09T07:51:52Z',
                    edges: [],
                    type: 'PROCESS_ACTIVITY'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-11-09T11:51:52Z',
                    edges: [],
                    type: 'FAILURE'
                }
            ]
        },
        {
            id: 'p-4',
            name: 'inactive-pod',
            startTime: '2019-11-09T05:51:52Z',
            inactive: true,
            numContainers: 0,
            events: []
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
                    type: 'POLICY_VIOLATION'
                },
                {
                    processId: 'process-2',
                    timestamp: '2019-12-09T07:51:52Z',
                    edges: [],
                    type: 'PROCESS_ACTIVITY'
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
                    type: 'PROCESS_ACTIVITY'
                },
                {
                    processId: 'process-4',
                    timestamp: '2019-12-09T09:51:52Z',
                    edges: [],
                    type: 'RESTART'
                }
            ]
        }
    ]
};
