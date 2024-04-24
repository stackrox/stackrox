import React from 'react';
import { Flex, Label, LabelGroup, Skeleton, Title } from '@patternfly/react-core';

import { ComplianceClusterOverallStats } from 'services/ComplianceEnhancedService';

import {
    getStatusCounts,
    calculateCompliancePercentage,
    getComplianceLabelGroupColor,
} from '../compliance.coverage.utils';

export type ClusterDetailsHeaderProps = {
    clusterStats: ComplianceClusterOverallStats | undefined;
    isLoading: boolean;
};

function ClusterDetailsHeader({ clusterStats, isLoading }: ClusterDetailsHeaderProps) {
    let passCount;
    let totalCount;
    let compliancePercentage;

    if (clusterStats) {
        ({ passCount, totalCount } = getStatusCounts(clusterStats.checkStats));
        compliancePercentage = calculateCompliancePercentage(passCount, totalCount);
    }

    return (
        <>
            <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsFlexStart' }}>
                <Title headingLevel="h1">
                    {isLoading ? (
                        <Skeleton
                            fontSize="2xl"
                            screenreaderText="Loading cluster name"
                            width="200px"
                        />
                    ) : (
                        clusterStats?.cluster.clusterName
                    )}
                </Title>
                <LabelGroup numLabels={1}>
                    <Label color={getComplianceLabelGroupColor(compliancePercentage)}>
                        {isLoading ? (
                            <Skeleton
                                screenreaderText="Loading compliance percentage"
                                width="110px"
                            />
                        ) : (
                            `Compliance: ${compliancePercentage}%`
                        )}
                    </Label>
                </LabelGroup>
            </Flex>
        </>
    );
}

export default ClusterDetailsHeader;
