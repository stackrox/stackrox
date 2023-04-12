import React, { ReactNode } from 'react';
import {
    Bullseye,
    Divider,
    Flex,
    Grid,
    GridItem,
    PageSection,
    Pagination,
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

import useURLStringUnion from 'hooks/useURLStringUnion';
import useURLSearch from 'hooks/useURLSearch';
import useURLPagination from 'hooks/useURLPagination';
import useURLSort from 'hooks/useURLSort';
import { getHasSearchApplied, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import EmptyStateTemplate from 'Components/PatternFly/EmptyStateTemplate';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { ExclamationCircleIcon } from '@patternfly/react-icons';
import WorkloadTableToolbar from './WorkloadTableToolbar';
import BySeveritySummaryCard from './SummaryCards/BySeveritySummaryCard';
import CvesByStatusSummaryCard from './SummaryCards/CvesByStatusSummaryCard';
import SingleEntityVulnerabilitiesTable from './Tables/SingleEntityVulnerabilitiesTable';
import useImageVulnerabilities from './hooks/useImageVulnerabilities';
import { DynamicTableLabel } from './components/DynamicIcon';
import { getHiddenSeverities, parseQuerySearchFilter } from './searchUtils';
import { QuerySearchFilter, FixableStatus, cveStatusTabValues } from './types';

function getHiddenStatuses(querySearchFilter: QuerySearchFilter): Set<FixableStatus> {
    const hiddenStatuses = new Set<FixableStatus>([]);
    const fixableFilters = querySearchFilter?.Fixable ?? [];

    if (fixableFilters.length > 0) {
        if (!fixableFilters.includes('true')) {
            hiddenStatuses.add('Fixable');
        }

        if (!fixableFilters.includes('false')) {
            hiddenStatuses.add('Not fixable');
        }
    }

    return hiddenStatuses;
}

const defaultSortFields = ['CVE', 'Severity', 'Fixable'];

export type ImagePageVulnerabilitiesProps = {
    imageId: string;
};

function ImagePageVulnerabilities({ imageId }: ImagePageVulnerabilitiesProps) {
    const { searchFilter } = useURLSearch();
    const querySearchFilter = parseQuerySearchFilter(searchFilter);
    const { page, perPage, setPage, setPerPage } = useURLPagination(50);
    const { sortOption, getSortParams } = useURLSort({
        sortFields: defaultSortFields,
        defaultSortOption: {
            field: 'Severity',
            direction: 'desc',
        },
        onSort: () => setPage(1),
    });

    const pagination = {
        offset: (page - 1) * perPage,
        limit: perPage,
        sortOption,
    };
    const { data, previousData, loading, error } = useImageVulnerabilities(
        imageId,
        getRequestQueryStringForSearchFilter(querySearchFilter),
        pagination
    );

    const [activeTabKey, setActiveTabKey] = useURLStringUnion('cveStatus', cveStatusTabValues);

    const isFiltered = getHasSearchApplied(querySearchFilter);

    let mainContent: ReactNode | null = null;

    const vulnerabilityData = data ?? previousData;

    if (error) {
        mainContent = (
            <Bullseye>
                <EmptyStateTemplate
                    headingLevel="h2"
                    title={getAxiosErrorMessage(error)}
                    icon={ExclamationCircleIcon}
                    iconClassName="pf-u-danger-color-100"
                >
                    Adjust your filters and try again
                </EmptyStateTemplate>
            </Bullseye>
        );
    } else if (loading && !vulnerabilityData) {
        mainContent = (
            <Bullseye>
                <Spinner isSVG />
            </Bullseye>
        );
    } else if (vulnerabilityData) {
        const hiddenSeverities = getHiddenSeverities(querySearchFilter);
        const hiddenStatuses = getHiddenStatuses(querySearchFilter);
        const vulnCounter = vulnerabilityData.image.imageVulnerabilityCounter;
        const totalVulnerabilityCount = vulnCounter.all.total;
        const { critical, important, moderate, low } = vulnCounter;

        mainContent = (
            <>
                <div className="pf-u-px-lg pf-u-pb-lg">
                    <Grid hasGutter>
                        <GridItem sm={12} md={6} xl2={4}>
                            <BySeveritySummaryCard
                                title="CVEs by severity"
                                severityCounts={{
                                    CRITICAL_VULNERABILITY_SEVERITY: critical.total,
                                    IMPORTANT_VULNERABILITY_SEVERITY: important.total,
                                    MODERATE_VULNERABILITY_SEVERITY: moderate.total,
                                    LOW_VULNERABILITY_SEVERITY: low.total,
                                }}
                                hiddenSeverities={hiddenSeverities}
                            />
                        </GridItem>
                        <GridItem sm={12} md={6} xl2={4}>
                            <CvesByStatusSummaryCard
                                cveStatusCounts={vulnerabilityData.image.imageVulnerabilityCounter}
                                hiddenStatuses={hiddenStatuses}
                            />
                        </GridItem>
                    </Grid>
                </div>
                <Divider />
                <div className="pf-u-p-lg">
                    <Split className="pf-u-pb-lg pf-u-align-items-baseline">
                        <SplitItem isFilled>
                            <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                <Title headingLevel="h2">
                                    {pluralize(totalVulnerabilityCount, 'result', 'results')} found
                                </Title>
                                {isFiltered && <DynamicTableLabel />}
                            </Flex>
                        </SplitItem>
                        <SplitItem>
                            <Pagination
                                isCompact
                                itemCount={totalVulnerabilityCount}
                                page={page}
                                perPage={perPage}
                                onSetPage={(_, newPage) => setPage(newPage)}
                                onPerPageSelect={(_, newPerPage) => {
                                    if (totalVulnerabilityCount < (page - 1) * newPerPage) {
                                        setPage(1);
                                    }
                                    setPerPage(newPerPage);
                                }}
                            />
                        </SplitItem>
                    </Split>
                    <SingleEntityVulnerabilitiesTable
                        image={vulnerabilityData.image}
                        getSortParams={getSortParams}
                        isFiltered={isFiltered}
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

export default ImagePageVulnerabilities;
