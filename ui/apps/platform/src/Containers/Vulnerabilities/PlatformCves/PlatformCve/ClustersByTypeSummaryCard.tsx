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
    clusterCounts: ClustersByType['clusterCountByType'];
};

function ClustersByTypeSummaryCard({ clusterCounts }: ClustersByTypeSummaryCardProps) {
    const { generic, kubernetes, openshift, openshift4 } = clusterCounts;
    return (
        <Card isCompact isFlat isFullHeight>
            <CardTitle>Clusters by type</CardTitle>
            <CardBody>
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
            </CardBody>
        </Card>
    );
}

export default ClustersByTypeSummaryCard;
