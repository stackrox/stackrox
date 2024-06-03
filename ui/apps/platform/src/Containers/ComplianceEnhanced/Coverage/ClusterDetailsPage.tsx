import React, { useCallback } from 'react';
import { generatePath, useParams } from 'react-router-dom';
import {
    Alert,
    Breadcrumb,
    BreadcrumbItem,
    Divider,
    Flex,
    Label,
    LabelGroup,
    PageSection,
    Skeleton,
    Title,
} from '@patternfly/react-core';

import BreadcrumbItemLink from 'Components/BreadcrumbItemLink';
import PageTitle from 'Components/PageTitle';
import useRestQuery from 'hooks/useRestQuery';
import { getComplianceProfilesStats } from 'services/ComplianceResultsStatsService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    coverageProfileClustersPath,
    coverageClusterDetailsPath,
} from './compliance.coverage.routes';
import ProfilesToggleGroup from './ProfilesToggleGroup';

function ClusterDetailsPage() {
    const { clusterId, profileName } = useParams();

    const fetchProfilesStats = useCallback(
        () => getComplianceProfilesStats(clusterId),
        [clusterId]
    );
    const {
        data: clusterProfileData,
        loading: isLoadingClusterProfileData,
        error: clusterProfileDataError,
    } = useRestQuery(fetchProfilesStats);

    if (clusterProfileDataError) {
        return (
            <Alert
                variant="warning"
                title="Unable to fetch cluster profiles"
                component="div"
                isInline
            >
                {getAxiosErrorMessage(clusterProfileDataError)}
            </Alert>
        );
    }

    return (
        <>
            <PageTitle title="Compliance coverage - Cluster" />
            <PageSection variant="light" className="pf-v5-u-py-md">
                <Breadcrumb>
                    <BreadcrumbItem>Compliance coverage</BreadcrumbItem>
                    <BreadcrumbItemLink
                        to={generatePath(coverageProfileClustersPath, {
                            profileName,
                        })}
                    >
                        Clusters
                    </BreadcrumbItemLink>
                    <BreadcrumbItem isActive>
                        {isLoadingClusterProfileData ? (
                            <Skeleton screenreaderText="Loading cluster name" width="150px" />
                        ) : (
                            // TODO: placeholder until we get cluster name
                            clusterId
                        )}
                    </BreadcrumbItem>
                </Breadcrumb>
            </PageSection>
            <Divider component="div" />
            <PageSection variant="light">
                <Flex
                    direction={{ default: 'column' }}
                    alignItems={{ default: 'alignItemsFlexStart' }}
                >
                    <Title headingLevel="h1" className="pf-v5-u-w-100">
                        {isLoadingClusterProfileData ? (
                            <Skeleton fontSize="2xl" screenreaderText="Loading cluster name" />
                        ) : (
                            // TODO: placeholder until we get cluster name
                            clusterId
                        )}
                    </Title>
                    <LabelGroup numLabels={1}>
                        <Label>
                            {isLoadingClusterProfileData ? (
                                <Skeleton
                                    screenreaderText="Loading number of profiles scanned on cluster"
                                    width="135px"
                                />
                            ) : (
                                `Scanned by: ${clusterProfileData?.totalCount} profiles`
                            )}
                        </Label>
                    </LabelGroup>
                </Flex>
            </PageSection>
            <Divider component="div" />
            <PageSection>
                <ProfilesToggleGroup
                    profiles={clusterProfileData?.scanStats || []}
                    route={coverageClusterDetailsPath.replace(':clusterId', clusterId)}
                />
            </PageSection>
        </>
    );
}

export default ClusterDetailsPage;
