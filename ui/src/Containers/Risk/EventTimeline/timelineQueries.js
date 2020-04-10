import gql from 'graphql-tag';

export const DEPLOYMENT_OVERVIEW_FRAGMENT = gql`
    fragment deploymentOverviewFields on Deployment {
        numPolicyViolations: failingPolicyCount
        numProcessActivities: processActivityCount
        numRestarts: containerRestartCount
        numTerminations: containerTerminationCount
        numTotalPods: podCount
    }
`;

export const POD_FRAGMENT = gql`
    fragment podFields on Pod {
        id
        name
        startTime: started
        liveInstances {
            finished
            started
        }
    }
`;

export const GET_EVENT_TIMELINE_OVERVIEW = gql`
    query getEventTimelineOverview($deploymentId: ID!) {
        deployment(id: $deploymentId) {
            ...deploymentOverviewFields
        }
    }
    ${DEPLOYMENT_OVERVIEW_FRAGMENT}
`;

export const GET_DEPLOYMENT_EVENT_TIMELINE = gql`
    query getDeploymentEventTimeline(
        $deploymentId: ID!
        $podsQuery: String
        $pagination: Pagination
    ) {
        deployment(id: $deploymentId) {
            ...deploymentOverviewFields
        }
        pods(query: $podsQuery, pagination: $pagination) {
            ...podFields
            events {
                type: __typename
                ... on PolicyViolationEvent {
                    id
                    name
                    timestamp
                }
                ... on ProcessActivityEvent {
                    id
                    name
                    timestamp
                    uid
                }
                ... on ContainerRestartEvent {
                    id
                    name
                    timestamp
                }
                ... on ContainerTerminationEvent {
                    id
                    name
                    timestamp
                    exitCode
                    reason
                }
            }
        }
    }
    ${DEPLOYMENT_OVERVIEW_FRAGMENT}
    ${POD_FRAGMENT}
`;

export const GET_POD_EVENT_TIMELINE = gql`
    query getPodEventTimeline($podId: ID!, $containersQuery: String) {
        pod(id: $podId) {
            ...podFields
        }
        containers: containerInstances(query: $containersQuery) {
            instanceId {
                id
            }
            name: containerName
            startTime: started
            events {
                type: __typename
                ... on PolicyViolationEvent {
                    id
                    name
                    timestamp
                }
                ... on ProcessActivityEvent {
                    id
                    name
                    timestamp
                    uid
                }
                ... on ContainerRestartEvent {
                    id
                    name
                    timestamp
                }
                ... on ContainerTerminationEvent {
                    id
                    name
                    timestamp
                    exitCode
                    reason
                }
            }
        }
    }
    ${POD_FRAGMENT}
`;
