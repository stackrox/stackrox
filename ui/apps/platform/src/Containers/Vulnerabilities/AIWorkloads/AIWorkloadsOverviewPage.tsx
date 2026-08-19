import { useCallback } from 'react';
import {
    Bullseye,
    Content,
    Divider,
    PageSection,
    Spinner,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
} from '@patternfly/react-core';

import useRestQuery from 'hooks/useRestQuery';
import EmptyStateTemplate from 'Components/EmptyStateTemplate';
import { listAIWorkloads } from 'services/AIWorkloadService';
import AIWorkloadsTable from './AIWorkloadsTable';

function AIWorkloadsOverviewPage() {
    const fetchWorkloads = useCallback(() => listAIWorkloads(), []);

    const { data, isLoading, error } = useRestQuery(fetchWorkloads);

    const workloads = data?.aiWorkloads ?? [];

    return (
        <>
            <PageSection>
                <Title headingLevel="h1">AI Workloads</Title>
                <Content>View and manage AI workloads detected on RHOAI-enabled clusters.</Content>
            </PageSection>
            <Divider />
            <PageSection>
                <Toolbar>
                    <ToolbarContent>
                        <ToolbarItem>
                            <Content>{data ? `${workloads.length} results found` : ''}</Content>
                        </ToolbarItem>
                    </ToolbarContent>
                </Toolbar>
                {isLoading && (
                    <Bullseye>
                        <Spinner />
                    </Bullseye>
                )}
                {error && (
                    <EmptyStateTemplate title="Unable to load AI workloads" headingLevel="h2">
                        {error.message}
                    </EmptyStateTemplate>
                )}
                {workloads.length > 0 && <AIWorkloadsTable workloads={workloads} />}
                {data && workloads.length === 0 && (
                    <EmptyStateTemplate title="No AI workloads found" headingLevel="h2">
                        No AI workloads have been detected on connected clusters. Deploy a model on
                        an RHOAI-enabled cluster to see it here.
                    </EmptyStateTemplate>
                )}
            </PageSection>
        </>
    );
}

export default AIWorkloadsOverviewPage;
