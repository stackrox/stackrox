import React, { ReactElement } from 'react';
import dateFns from 'date-fns';
import { DescriptionList } from '@patternfly/react-core';

import dateTimeFormat from 'constants/dateTimeFormat';
import DescriptionListItem from 'Components/DescriptionListItem';
import { AlertDeployment } from 'types/alert.proto';
import { Deployment } from 'types/deployment.proto';

import FlatObjectDescriptionList from './FlatObjectDescriptionList';

export type DeploymentOverviewProps = {
    alertDeployment: AlertDeployment;
    deployment?: Deployment;
};

function DeploymentOverview({
    alertDeployment,
    deployment,
}: DeploymentOverviewProps): ReactElement {
    return (
        <DescriptionList isCompact isHorizontal>
            <DescriptionListItem term="Deployment ID" desc={alertDeployment.id} />
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
