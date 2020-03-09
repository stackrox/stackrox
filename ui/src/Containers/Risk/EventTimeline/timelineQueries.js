import gql from 'graphql-tag';

export const OVERVIEW_FRAGMENT = gql`
    fragment overviewFields on Deployment {
        numPolicyViolations: failingPolicyCount
        numProcessActivities: processActivityCount
        numRestarts: containerRestartCount
        numTerminations: containerTerminationCount
        numTotalPods: podCount
    }
`;

export const GET_EVENT_TIMELINE_OVERVIEW = gql`
    query getEventTimelineOverview($deploymentId: ID!) {
        deployment(id: $deploymentId) {
            ...overviewFields
        }
    }
    ${OVERVIEW_FRAGMENT}
`;

export const GET_DEPLOYMENT_EVENT_TIMELINE = gql`
    query getDeploymentEventTimeline(
        $deploymentId: ID!
        $podsQuery: String
        $pagination: Pagination
    ) {
        deployment(id: $deploymentId) {
            ...overviewFields
        }
        pods(query: $podsQuery, pagination: $pagination) {
            id
            name
            startTime
            inactive
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
    ${OVERVIEW_FRAGMENT}
`;
