import React from 'react';
import { gql } from '@apollo/client';
import { Card, CardTitle, CardBody, Grid, GridItem } from '@patternfly/react-core';

export const clustersByTypeFragment = gql`
    fragment ClustersByType on PlatformCVECore {
        clusterCountByType {
            generic
            kubernetes
            openshift
            openshift4
        }
    }
`;

export type ClustersByType = {
    clusterCountByType: {
        generic: number;
        kubernetes: number;
        openshift: number;
        openshift4: number;
    };
};

export type ClustersByTypeSummaryCardProps = {
    clusterCounts?: ClustersByType['clusterCountByType'];
};

function ClustersByTypeSummaryCard({ clusterCounts }: ClustersByTypeSummaryCardProps) {
    const { generic = 0, kubernetes = 0, openshift = 0, openshift4 = 0 } = clusterCounts ?? {};
    const totalCount = generic + kubernetes + openshift + openshift4;

    return (
        <Card isCompact isFlat isFullHeight>
            <CardTitle>Clusters by type</CardTitle>
            <CardBody>
                {totalCount > 0 ? (
                    <Grid>
                        {generic > 0 && (
                            <GridItem span={12} className="pf-v5-u-pt-xs">
                                {generic} Generic
                            </GridItem>
                        )}
                        {kubernetes > 0 && (
                            <GridItem span={12} className="pf-v5-u-pt-xs">
                                {kubernetes} Kubernetes
                            </GridItem>
                        )}
                        {openshift + openshift4 > 0 && (
                            <GridItem span={12} className="pf-v5-u-pt-xs">
                                {openshift + openshift4} OpenShift
                            </GridItem>
                        )}
                    </Grid>
                ) : (
                    <Grid>
                        <GridItem span={12} className="pf-v5-u-pt-xs">
                            No affected clusters found
                        </GridItem>
                    </Grid>
                )}
            </CardBody>
        </Card>
    );
}

export default ClustersByTypeSummaryCard;
