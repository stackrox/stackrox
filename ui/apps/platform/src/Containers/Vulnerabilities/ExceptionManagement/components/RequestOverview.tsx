import React from 'react';
import {
    DescriptionList,
    DescriptionListDescription,
    DescriptionListGroup,
    DescriptionListTerm,
    Flex,
    PageSection,
    Title,
} from '@patternfly/react-core';
import { VulnerabilityException } from 'services/VulnerabilityExceptionService';
import {
    RequestComment,
    RequestComments,
    RequestContext,
    RequestCreatedAt,
    RequestExpires,
    RequestScope,
    RequestedAction,
} from './ExceptionRequestTableCells';

export type RequestOverviewProps = {
    exception: VulnerabilityException;
    context: RequestContext;
};

function RequestOverview({ exception, context }: RequestOverviewProps) {
    // There will always be at least 1 comment
    const latestComment = exception.comments.at(-1);

    return (
        <PageSection variant="light">
            <Flex direction={{ default: 'column' }}>
                <Title headingLevel="h2">Overview</Title>
                <DescriptionList className="vulnerability-exception-request-overview">
                    <DescriptionListGroup>
                        <DescriptionListTerm>Requestor</DescriptionListTerm>
                        <DescriptionListDescription>
                            {exception.requester?.name || '-'}
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Requested action</DescriptionListTerm>
                        <DescriptionListDescription>
                            <RequestedAction exception={exception} context={context} />
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Requested</DescriptionListTerm>
                        <DescriptionListDescription>
                            <RequestCreatedAt createdAt={exception.createdAt} />
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Expires</DescriptionListTerm>
                        <DescriptionListDescription>
                            <RequestExpires exception={exception} context={context} />
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Scope</DescriptionListTerm>
                        <DescriptionListDescription>
                            <RequestScope scope={exception.scope} />
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    <DescriptionListGroup>
                        <DescriptionListTerm>Comments</DescriptionListTerm>
                        <DescriptionListDescription>
                            <RequestComments comments={exception.comments} />
                        </DescriptionListDescription>
                    </DescriptionListGroup>
                    {latestComment && (
                        <DescriptionListGroup>
                            <DescriptionListTerm>Latest comment</DescriptionListTerm>
                            <DescriptionListDescription>
                                <RequestComment comment={latestComment} />
                            </DescriptionListDescription>
                        </DescriptionListGroup>
                    )}
                </DescriptionList>
            </Flex>
        </PageSection>
    );
}

export default RequestOverview;
