import {
    Alert,
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Spinner,
} from '@patternfly/react-core';
import { gql, useQuery } from '@apollo/client';

import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

const GET_EVENT_TIMELINE_OVERVIEW = gql`
    query getEventTimelineOverview($deploymentId: ID!) {
        deployment(id: $deploymentId) {
            numPolicyViolations: failingRuntimePolicyCount
            numProcessActivities: processActivityCount
            numRestarts: containerRestartCount
            numTerminations: containerTerminationCount
        }
    }
`;

export type EventTimelineOverviewProps = {
    deploymentId: string;
};

function EventTimelineOverview({ deploymentId }: EventTimelineOverviewProps) {
    const { loading, error, data } = useQuery(GET_EVENT_TIMELINE_OVERVIEW, {
        variables: { deploymentId },
    });

    if (loading) {
        return <Spinner />;
    }

    if (error) {
        return (
            <Alert variant="warning" title="Unable to fetch event timeline" component="p" isInline>
                {getAxiosErrorMessage(error)}
            </Alert>
        );
    }

    // data fetching with "deploymentId" will happen here...
    const { numPolicyViolations, numProcessActivities, numRestarts, numTerminations } =
        data.deployment;

    return (
        <DescriptionList
            isCompact
            isHorizontal
            horizontalTermWidthModifier={{
                default: '20ch',
            }}
        >
            <DescriptionListGroup>
                <DescriptionListTerm>Policy violations</DescriptionListTerm>
                <DescriptionListDescription>{numPolicyViolations}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Process activities</DescriptionListTerm>
                <DescriptionListDescription>{numProcessActivities}</DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
                <DescriptionListTerm>Restarts + Terminations</DescriptionListTerm>
                <DescriptionListDescription>
                    {numRestarts + numTerminations}
                </DescriptionListDescription>
            </DescriptionListGroup>
        </DescriptionList>
    );
}

export default EventTimelineOverview;
