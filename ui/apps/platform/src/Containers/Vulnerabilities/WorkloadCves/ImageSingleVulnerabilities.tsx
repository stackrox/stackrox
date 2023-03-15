import React, { ReactNode } from 'react';
import {
    Bullseye,
    Divider,
    EmptyState,
    EmptyStateBody,
    EmptyStateIcon,
    EmptyStateVariant,
    Flex,
    Grid,
    GridItem,
    Label,
    PageSection,
    pluralize,
    Spinner,
    Split,
    SplitItem,
    Tab,
    TabTitleText,
    Tabs,
    TabsComponent,
    Text,
    Title,
} from '@patternfly/react-core';
import { ExclamationCircleIcon, InfoCircleIcon } from '@patternfly/react-icons';

import { VulnerabilitySeverity } from 'types/cve.proto';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLSearch from 'hooks/useURLSearch';
import { getHasSearchApplied } from 'utils/searchUtils';
import { cveStatusTabValues, FixableStatus } from './types';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import BySeveritySummaryCard from './SummaryCards/BySeveritySummaryCard';
import CvesByStatusSummaryCard from './SummaryCards/CvesByStatusSummaryCard';
import SingleEntityVulnerabilitiesTable from './Tables/SingleEntityVulnerabilitiesTable';
import useImageVulnerabilities, {
    ImageVulnerabilitiesResponse,
} from './hooks/useImageVulnerabilities';

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

export type ImageSingleVulnerabilitiesProps = {
    imageId: string;
};

function ImageSingleVulnerabilities({ imageId }: ImageSingleVulnerabilitiesProps) {
    const { searchFilter } = useURLSearch();
    // TODO Still need to properly integrate search filter with query
    const { data, loading, error } = useImageVulnerabilities(imageId, {});

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('cveStatus', cveStatusTabValues);

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
            <>
                <div className="pf-u-px-lg pf-u-pb-lg">
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
                </div>
                <Divider />
                <div className="pf-u-p-lg">
                    <Split className="pf-u-pb-lg">
                        <SplitItem isFilled>
                            <Flex alignContent={{ default: 'alignContentCenter' }}>
                                <Title headingLevel="h2">
                                    {pluralize(
                                        data.image.imageVulnerabilities.length,
                                        'result',
                                        'results'
                                    )}{' '}
                                    found
                                </Title>
                                {getHasSearchApplied(searchFilter) && (
                                    <Label isCompact color="blue" icon={<InfoCircleIcon />}>
                                        Filtered view
                                    </Label>
                                )}
                            </Flex>
                        </SplitItem>
                        <SplitItem>TODO Pagination</SplitItem>
                    </Split>
                    <SingleEntityVulnerabilitiesTable
                        imageVulnerabilities={data.image.imageVulnerabilities}
                    />
                </div>
            </>
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
                    onSelect={(e, key) => setActiveTabKey(key)}
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
                        <div className="pf-u-px-sm pf-u-background-color-100">
                            <WorkloadTableToolbar />
                        </div>
                        <div className="pf-u-flex-grow-1 pf-u-background-color-100">
                            {mainContent}
                        </div>
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
