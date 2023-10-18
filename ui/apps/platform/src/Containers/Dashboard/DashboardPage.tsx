import React, { CSSProperties } from 'react';
import { Divider, Flex, FlexItem, Gallery, PageSection, Text, Title } from '@patternfly/react-core';

import usePermissions from 'hooks/usePermissions';

import SummaryCounts from './SummaryCounts';
import ScopeBar from './ScopeBar';

import ImagesAtMostRisk from './Widgets/ImagesAtMostRisk';
import ViolationsByPolicyCategory from './Widgets/ViolationsByPolicyCategory';
import DeploymentsAtMostRisk from './Widgets/DeploymentsAtMostRisk';
import AgingImages from './Widgets/AgingImages';
import ViolationsByPolicySeverity from './Widgets/ViolationsByPolicySeverity';
import ComplianceLevelsByStandard from './Widgets/ComplianceLevelsByStandard';

// This value is an estimate of the minimum size the widgets need to be to
// ensure the heading and options do not wrap and break layout.
const minWidgetWidth = 510;

function DashboardPage() {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForAlert = hasReadAccess('Alert');
    const hasReadAccessForCluster = hasReadAccess('Cluster');
    const hasReadAccessForCompliance = hasReadAccess('Compliance');
    const hasReadAccessForDeployment = hasReadAccess('Deployment');
    const hasReadAccessForImage = hasReadAccess('Image');
    const hasReadAccessForNamespace = hasReadAccess('Namespace');
    const hasReadAccessForNode = hasReadAccess('Node');
    const hasReadAccessForSecret = hasReadAccess('Secret');

    const hasReadAccessForSummaryCounts =
        hasReadAccessForAlert ||
        hasReadAccessForCluster ||
        hasReadAccessForDeployment ||
        hasReadAccessForImage ||
        hasReadAccessForNode ||
        hasReadAccessForSecret;

    return (
        <>
            {hasReadAccessForSummaryCounts && (
                <>
                    <PageSection variant="light" padding={{ default: 'noPadding' }}>
                        <SummaryCounts
                            Cluster={hasReadAccessForCluster}
                            Node={hasReadAccessForNode}
                            Violation={hasReadAccessForAlert}
                            Deployment={hasReadAccessForDeployment}
                            Image={hasReadAccessForImage}
                            Secret={hasReadAccessForSecret}
                        />
                    </PageSection>
                    <Divider component="div" />
                </>
            )}
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column', lg: 'row' }}
                    alignItems={{ default: 'alignItemsFlexStart', lg: 'alignItemsCenter' }}
                >
                    <FlexItem>
                        <Title headingLevel="h1">Dashboard</Title>
                        <Text>Review security metrics across all or select resources</Text>
                    </FlexItem>
                    {hasReadAccessForCluster && hasReadAccessForNamespace && (
                        <FlexItem
                            grow={{ default: 'grow' }}
                            className="pf-u-display-flex pf-u-justify-content-flex-end"
                        >
                            <ScopeBar />
                        </FlexItem>
                    )}
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <Gallery
                    style={
                        {
                            // Ensure the grid has never grows large enough to show 4 columns
                            maxWidth: `calc(calc(${minWidgetWidth}px * 4) + calc(var(--pf-l-gallery--m-gutter--GridGap) * 3) - 1px)`,
                            // Ensure the grid gap matches that of the outside padding of the containing PageSection
                            '--pf-l-gallery--m-gutter--GridGap':
                                'var(--pf-c-page__main-section--PaddingTop)',
                        } as CSSProperties
                    }
                    hasGutter
                    minWidths={{ default: `${minWidgetWidth}px` }}
                >
                    {hasReadAccessForAlert && <ViolationsByPolicySeverity />}
                    {hasReadAccessForImage && <ImagesAtMostRisk />}
                    {hasReadAccessForDeployment && <DeploymentsAtMostRisk />}
                    {hasReadAccessForImage && <AgingImages />}
                    {hasReadAccessForAlert && <ViolationsByPolicyCategory />}
                    {hasReadAccessForCompliance && <ComplianceLevelsByStandard />}
                </Gallery>
            </PageSection>
        </>
    );
}

export default DashboardPage;
