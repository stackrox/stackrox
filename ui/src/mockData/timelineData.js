const pods = [
    {
        id: 'p-1',
        name: 'hello-world',
        startTime: '2019-12-12T00:00:00Z',
        inactive: false,
        numContainers: 3,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-12-12T01:00:00Z',
                edges: [],
                type: 'POLICY_VIOLATION'
            },
            {
                processId: 'process-2',
                timestamp: '2019-12-12T02:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            },
            {
                processId: 'process-3',
                timestamp: '2019-12-12T04:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            },
            {
                processId: 'process-4',
                timestamp: '2019-12-12T05:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            },
            {
                processId: 'process-5',
                timestamp: '2019-12-12T09:00:00Z',
                edges: [],
                type: 'RESTART'
            }
        ]
    },
    {
        id: 'p-2',
        name: 'dr-seuss',
        startTime: '2019-12-16T00:00:00Z',
        inactive: false,
        numContainers: 0,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-12-16T01:00:00Z',
                edges: [],
                type: 'POLICY_VIOLATION'
            },
            {
                processId: 'process-2',
                timestamp: '2019-12-16T08:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            },
            {
                processId: 'process-3',
                timestamp: '2019-12-16T07:00:00Z',
                edges: [],
                type: 'POLICY_VIOLATION'
            },
            {
                processId: 'process-4',
                timestamp: '2019-12-16T06:00:00Z',
                edges: [],
                type: 'RESTART'
            }
        ]
    },
    {
        id: 'p-3',
        name: 'two-peas-in-a-pod',
        startTime: '2019-12-13T00:00:00Z',
        inactive: false,
        numContainers: 1,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-12-13T04:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            },
            {
                processId: 'process-2',
                timestamp: '2019-12-13T03:00:00Z',
                edges: [],
                type: 'FAILURE'
            },
            {
                processId: 'process-3',
                timestamp: '2019-12-13T02:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            },
            {
                processId: 'process-4',
                timestamp: '2019-12-13T06:00:00Z',
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
    },
    {
        id: 'p-5',
        name: 'ivan',
        startTime: '2019-11-09T00:00:00Z',
        inactive: false,
        numContainers: 0,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-11-09T05:00:00Z',
                edges: [],
                type: 'FAILURE'
            }
        ]
    },
    {
        id: 'p-6',
        name: 'alan-roy-jr',
        startTime: '2019-11-09T00:00:00Z',
        inactive: false,
        numContainers: 0,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-11-09T05:00:00Z',
                edges: [],
                type: 'FAILURE'
            },
            {
                processId: 'process-2',
                timestamp: '2019-11-09T03:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            }
        ]
    },
    {
        id: 'p-7',
        name: 'poderick',
        startTime: '2019-11-09T00:00:00Z',
        inactive: false,
        numContainers: 1,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-11-09T03:00:00Z',
                edges: [],
                type: 'FAILURE'
            },
            {
                processId: 'process-2',
                timestamp: '2019-11-09T08:00:00Z',
                edges: [],
                type: 'PROCESS_ACTIVITY'
            }
        ]
    },
    {
        id: 'p-8',
        name: 'poderella',
        startTime: '2019-11-09T00:00:00Z',
        inactive: false,
        numContainers: 2,
        events: [
            {
                processId: 'process-1',
                timestamp: '2019-11-09T05:00:00Z',
                edges: [],
                type: 'FAILURE'
            },
            {
                processId: 'process-2',
                timestamp: '2019-11-09T08:00:00Z',
                edges: [],
                type: 'FAILURE'
            }
        ]
    },
    {
        id: 'p-9',
        name: 'inactive-pod-2',
        startTime: '2019-11-09T00:00:00Z',
        inactive: true,
        numContainers: 0,
        events: []
    },
    {
        id: 'p-10',
        name: 'inactive-pod-3',
        startTime: '2019-11-09T00:00:00Z',
        inactive: true,
        numContainers: 0,
        events: []
    }
];

const {
    POLICY_VIOLATION: numPolicyViolations,
    PROCESS_ACTIVITY: numProcessActivities,
    RESTART: numRestarts,
    FAILURE: numFailures
} = pods.reduce(
    (acc, curr) => {
        curr.events.forEach(event => {
            acc[event.type] += 1;
        });
        return acc;
    },
    { POLICY_VIOLATION: 0, PROCESS_ACTIVITY: 0, RESTART: 0, FAILURE: 0 }
);

export const overviewData = {
    deployment: {
        numPolicyViolations,
        numProcessActivities,
        numRestarts,
        numFailures,
        numTotalPods: pods.length
    }
};

export const podsData = {
    ...overviewData,
    pods
};

const containers = [
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
];

export const getPodAndContainersByPodId = podId => {
    const { id, name, startTime, inactive } = podsData.pods.find(datum => datum.id === podId);
    return {
        pod: {
            id,
            name,
            startTime,
            inactive
        },
        containers
    };
};
