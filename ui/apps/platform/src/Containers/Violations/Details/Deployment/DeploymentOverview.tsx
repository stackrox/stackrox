import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import dateFns from 'date-fns';
import { DescriptionList } from '@patternfly/react-core';

import dateTimeFormat from 'constants/dateTimeFormat';
import DescriptionListItem from 'Components/DescriptionListItem';
import {
    vulnerabilitiesPlatformPath,
    vulnerabilitiesUserWorkloadsPath,
    vulnerabilitiesWorkloadCvesPath,
} from 'routePaths';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { AlertDeployment } from 'types/alert.proto';
import { Deployment } from 'types/deployment.proto';

import FlatObjectDescriptionList from './FlatObjectDescriptionList';

export type DeploymentOverviewProps = {
    alertDeployment: AlertDeployment;
    deployment: Deployment | null;
};

function DeploymentOverview({
    alertDeployment,
    deployment,
}: DeploymentOverviewProps): ReactElement {
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const hasPlatformWorkloadCveLink = deployment && deployment.platformComponent;
    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListItem
                term="Deployment ID"
                desc={
                    <Link
                        to={
                            // eslint-disable-next-line no-nested-ternary
                            !isFeatureFlagEnabled('ROX_PLATFORM_CVE_SPLIT')
                                ? `${vulnerabilitiesWorkloadCvesPath}/deployments/${alertDeployment.id}`
                                : hasPlatformWorkloadCveLink
                                  ? `${vulnerabilitiesPlatformPath}/deployments/${alertDeployment.id}`
                                  : `${vulnerabilitiesUserWorkloadsPath}/deployments/${alertDeployment.id}`
                        }
                    >
                        {alertDeployment.id}
                    </Link>
                }
            />
            <DescriptionListItem term="Deployment name" desc={alertDeployment.name} />
            <DescriptionListItem term="Deployment type" desc={alertDeployment.type} />
            <DescriptionListItem term="Cluster" desc={alertDeployment.clusterName} />
            <DescriptionListItem term="Namespace" desc={alertDeployment.namespace} />
            {deployment && (
                <>
                    <DescriptionListItem term="Replicas" desc={deployment.replicas} />
                    <DescriptionListItem
                        term="Created"
                        desc={
                            deployment.created
                                ? dateFns.format(deployment.created, dateTimeFormat)
                                : 'not available'
                        }
                    />
                    <DescriptionListItem
                        term="Labels"
                        desc={<FlatObjectDescriptionList data={deployment.labels} />}
                    />
                    <DescriptionListItem
                        term="Annotations"
                        desc={<FlatObjectDescriptionList data={deployment.annotations} />}
                    />
                    <DescriptionListItem term="Service account" desc={deployment.serviceAccount} />
                    {Array.isArray(deployment.imagePullSecrets) &&
                        deployment.imagePullSecrets.length > 0 && (
                            <DescriptionListItem
                                term="Image pull secrets"
                                desc={deployment.imagePullSecrets.join(', ')}
                            />
                        )}
                </>
            )}
        </DescriptionList>
    );
}

export default DeploymentOverview;
