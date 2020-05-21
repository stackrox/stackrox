import gql from 'graphql-tag';

export const DEPLOYMENT_OVERVIEW_FRAGMENT = gql`
    fragment deploymentOverviewFields on Deployment {
        name
        numPolicyViolations: failingRuntimePolicyCount
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
        containerCount
    }
`;

export const POLICY_VIOLATION_EVENT_FRAGMENT = gql`
    fragment policyViolationEventFields on PolicyViolationEvent {
        type: __typename
        id
        name
        timestamp
    }
`;

export const PROCESS_ACTIVITY_EVENT_FRAGMENT = gql`
    fragment processActivityEventFields on ProcessActivityEvent {
        type: __typename
        id
        name
        args
        timestamp
        uid
        parentName
        parentUid
        whitelisted
    }
`;

export const RESTART_EVENT_FRAGMENT = gql`
    fragment restartEventFields on ContainerRestartEvent {
        type: __typename
        id
        name
        timestamp
    }
`;

export const TERMINATION_EVENT_FRAGMENT = gql`
    fragment terminationEventFields on ContainerTerminationEvent {
        type: __typename
        id
        name
        timestamp
        exitCode
        reason
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
                ...policyViolationEventFields
                ...processActivityEventFields
                ...restartEventFields
                ...terminationEventFields
            }
        }
    }
    ${DEPLOYMENT_OVERVIEW_FRAGMENT}
    ${POD_FRAGMENT}
    ${POLICY_VIOLATION_EVENT_FRAGMENT}
    ${PROCESS_ACTIVITY_EVENT_FRAGMENT}
    ${RESTART_EVENT_FRAGMENT}
    ${TERMINATION_EVENT_FRAGMENT}
`;

export const GET_POD_EVENT_TIMELINE = gql`
    query getPodEventTimeline($podId: ID!, $containersQuery: String) {
        pod(id: $podId) {
            ...podFields
        }
        containers: groupedContainerInstances(query: $containersQuery) {
            id
            name
            startTime
            events {
                ...policyViolationEventFields
                ...processActivityEventFields
                ...restartEventFields
                ...terminationEventFields
            }
        }
    }
    ${POD_FRAGMENT}
    ${POLICY_VIOLATION_EVENT_FRAGMENT}
    ${PROCESS_ACTIVITY_EVENT_FRAGMENT}
    ${RESTART_EVENT_FRAGMENT}
    ${TERMINATION_EVENT_FRAGMENT}
`;
