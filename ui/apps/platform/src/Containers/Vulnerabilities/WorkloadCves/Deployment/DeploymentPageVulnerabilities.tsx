import React from 'react';
import {
    Divider,
    Flex,
    PageSection,
    Pagination,
    pluralize,
    Split,
    SplitItem,
    Text,
    Title,
} from '@patternfly/react-core';

import useURLPagination from 'hooks/useURLPagination';
import { DynamicTableLabel } from '../components/DynamicIcon';
import WorkloadTableToolbar from '../components/WorkloadTableToolbar';
import CveStatusTabs, {
    DeferredCvesTab,
    FalsePositiveCvesTab,
    ObservedCvesTab,
} from '../components/CveStatusTabs';

export type DeploymentPageVulnerabilitiesProps = Record<string, never>;

function DeploymentPageVulnerabilities() {
    const { page, setPage, perPage, setPerPage } = useURLPagination(20);

    const totalVulnerabilityCount = 0;
    const isFiltered = false;

    return (
        <>
            <PageSection component="div" variant="light" className="pf-u-py-md pf-u-px-xl">
                <Text>
                    Review and triage vulnerability data scanned for images within this deployment
                </Text>
            </PageSection>
            <Divider component="div" />
            <PageSection
                className="pf-u-display-flex pf-u-flex-direction-column pf-u-flex-grow-1"
                component="div"
            >
                <CveStatusTabs isBox>
                    <ObservedCvesTab>
                        <div className="pf-u-px-sm pf-u-background-color-100">
                            <WorkloadTableToolbar />
                        </div>
                        <div className="pf-u-flex-grow-1 pf-u-background-color-100">
                            <div className="pf-u-px-lg pf-u-pb-lg">Summary Cards</div>
                            <Divider />
                            <div className="pf-u-p-lg">
                                <Split className="pf-u-pb-lg pf-u-align-items-baseline">
                                    <SplitItem isFilled>
                                        <Flex alignItems={{ default: 'alignItemsCenter' }}>
                                            <Title headingLevel="h2">
                                                {pluralize(
                                                    totalVulnerabilityCount,
                                                    'result',
                                                    'results'
                                                )}{' '}
                                                found
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
                                                if (
                                                    totalVulnerabilityCount <
                                                    (page - 1) * newPerPage
                                                ) {
                                                    setPage(1);
                                                }
                                                setPerPage(newPerPage);
                                            }}
                                        />
                                    </SplitItem>
                                </Split>
                                TODO Table
                            </div>
                        </div>
                    </ObservedCvesTab>
                    <DeferredCvesTab isDisabled />
                    <FalsePositiveCvesTab isDisabled />
                </CveStatusTabs>
            </PageSection>
        </>
    );
}

export default DeploymentPageVulnerabilities;
