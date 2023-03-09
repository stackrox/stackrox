import React, { ReactNode } from 'react';
import {
    Bullseye,
    Divider,
    EmptyState,
    EmptyStateBody,
    EmptyStateIcon,
    EmptyStateVariant,
    Grid,
    GridItem,
    PageSection,
    Spinner,
    Tab,
    TabTitleText,
    Tabs,
    TabsComponent,
    TabsProps,
    Text,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import { gql, useQuery } from '@apollo/client';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { FixableStatus, isValidCveStatusTab } from './types';
import useCveStatusTabParameter from './hooks/useCveStatusTabParameter';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import BySeveritySummaryCard from './SummaryCards/BySeveritySummaryCard';
import CvesByStatusSummaryCard from './SummaryCards/CvesByStatusSummaryCard';

export type ImageVulnerabilitiesVariables = {
    id: string;
};

export type ImageVulnerabilitiesResponse = {
    image: {
        imageVulnerabilities: {
            severity: string;
            isFixable: boolean;
        }[];
    };
};

function severityCountsFromImageVulnerabilities(
    imageVulnerabilities: ImageVulnerabilitiesResponse['image']['imageVulnerabilities']
): Record<VulnerabilitySeverity, number> {
    const severityCounts = {
        LOW_VULNERABILITY_SEVERITY: 0,
        MODERATE_VULNERABILITY_SEVERITY: 0,
        IMPORTANT_VULNERABILITY_SEVERITY: 0,
        CRITICAL_VULNERABILITY_SEVERITY: 0,
    };

    imageVulnerabilities.forEach(({ severity }) => {
        severityCounts[severity] += 1;
    });

    return severityCounts;
}

function statusCountsFromImageVulnerabilities(
    imageVulnerabilities: ImageVulnerabilitiesResponse['image']['imageVulnerabilities']
): Record<FixableStatus, number> {
    const statusCounts = {
        Fixable: 0,
        'Not fixable': 0,
    };

    imageVulnerabilities.forEach(({ isFixable }) => {
        if (isFixable) {
            statusCounts.Fixable += 1;
        } else {
            statusCounts['Not fixable'] += 1;
        }
    });

    return statusCounts;
}

export const imageVulnerabilitiesQuery = gql`
    query getImageVulnerabilities($id: ID!) {
        image(id: $id) {
            id
            imageVulnerabilities {
                severity
                isFixable
            }
        }
    }
`;

export type ImageSingleVulnerabilitiesProps = {
    imageId: string;
};

function ImageSingleVulnerabilities({ imageId }: ImageSingleVulnerabilitiesProps) {
    // TODO Needs integration with URL search filter
    const { data, loading, error } = useQuery<
        ImageVulnerabilitiesResponse,
        ImageVulnerabilitiesVariables
    >(imageVulnerabilitiesQuery, {
        variables: { id: imageId },
    });

    const [activeTabKey, setActiveTabKey] = useCveStatusTabParameter();

    const handleTabClick: TabsProps['onSelect'] = (e, tabKey) => {
        if (isValidCveStatusTab(tabKey)) {
            setActiveTabKey(tabKey);
        }
    };

    let mainContent: ReactNode | null = null;

    if (error) {
        mainContent = (
            <Bullseye>
                <EmptyState variant={EmptyStateVariant.large}>
                    <EmptyStateIcon
                        className="pf-u-danger-color-100"
                        icon={ExclamationCircleIcon}
                    />
                    <Title headingLevel="h2">{getAxiosErrorMessage(error)}</Title>
                    <EmptyStateBody>Adjust your filters and try again</EmptyStateBody>
                </EmptyState>
            </Bullseye>
        );
    } else if (loading && !data) {
        mainContent = (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    } else if (data) {
        const vulnerabilities = data.image.imageVulnerabilities;
        const severityCounts = severityCountsFromImageVulnerabilities(vulnerabilities);
        const cveStatusCounts = statusCountsFromImageVulnerabilities(vulnerabilities);
        // TODO Integrate these with page search filters
        const hiddenSeverities = new Set<VulnerabilitySeverity>([]);
        const hiddenStatuses = new Set<FixableStatus>([]);

        mainContent = (
            <Grid hasGutter>
                <GridItem sm={12} md={6} xl2={4}>
                    <BySeveritySummaryCard
                        title="CVEs by severity"
                        severityCounts={severityCounts}
                        hiddenSeverities={hiddenSeverities}
                    />
                </GridItem>
                <GridItem sm={12} md={6} xl2={4}>
                    <CvesByStatusSummaryCard
                        cveStatusCounts={cveStatusCounts}
                        hiddenStatuses={hiddenStatuses}
                    />
                </GridItem>
            </Grid>
        );
    }

    return (
        <>
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>Review and triage vulnerability data scanned on this image</Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                component="div"
            >
                <Tabs
                    activeKey={activeTabKey}
                    onSelect={handleTabClick}
                    component={TabsComponent.nav}
                    mountOnEnter
                    unmountOnExit
                    isBox
                >
                    <Tab
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        eventKey="Observed"
                        title={<TabTitleText>Observed CVEs</TabTitleText>}
                    >
                        <PageSection variant="light" component="div" isFilled>
                            <WorkloadTableToolbar
                                // TODO: wire up the actual default filters in this component
                                defaultFilters={{
                                    Severity: ['Critical'],
                                    Fixable: ['Fixable'],
                                }}
                            />
                            {mainContent}
                        </PageSection>
                    </Tab>
                    <Tab
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        eventKey="Deferred"
                        title={<TabTitleText>Deferrals</TabTitleText>}
                        isDisabled
                    />
                    <Tab
                        className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                        eventKey="False Positive"
                        title={<TabTitleText>False positives</TabTitleText>}
                        isDisabled
                    />
                </Tabs>
            </PageSection>
        </>
    );
}

export default ImageSingleVulnerabilities;
