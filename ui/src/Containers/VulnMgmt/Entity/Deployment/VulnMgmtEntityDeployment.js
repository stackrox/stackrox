import React from 'react';
import { format } from 'date-fns';

import entityTypes from 'constants/entityTypes';
import dateTimeFormat from 'constants/dateTimeFormat';
import Loader from 'Components/Loader';
import PageNotFound from 'Components/PageNotFound';
import CollapsibleSection from 'Components/CollapsibleSection';
import RelatedEntity from 'Containers/ConfigManagement/Entity/widgets/RelatedEntity';
import RelatedEntityListCount from 'Containers/ConfigManagement/Entity/widgets/RelatedEntityListCount';
import Metadata from 'Containers/ConfigManagement/Entity/widgets/Metadata';
import isGQLLoading from 'utils/gqlLoading';
import { entityComponentPropTypes, entityComponentDefaultProps } from 'constants/entityPageProps';

const VulmMgmtDeployment = ({ loading, data }) => {
    // (1) still waiting for data
    if (isGQLLoading(loading, data)) return <Loader transparent />;

    // (2) no deployment with that ID
    if (!data || !data.deployment) return <PageNotFound resourceType={entityTypes.DEPLOYMENT} />;
    const { deployment: entity } = data;

    // (3) if we get this far, display the deployment

    // TODO: calculate this higher up and pass in
    // const overlay = !!entityId1;
    const overlay = false;

    // TODO handle entity lists here

    const {
        cluster,
        created,
        type,
        replicas,
        labels = [],
        annotations = [],
        namespace,
        namespaceId,
        serviceAccount,
        serviceAccountID,
        imageCount,
        secretCount
    } = entity;

    const metadataKeyValuePairs = [
        {
            key: 'Created',
            value: created ? format(created, dateTimeFormat) : 'N/A'
        },
        {
            key: 'Deployment Type',
            value: type
        },
        {
            key: 'Replicas',
            value: replicas
        }
    ];

    return (
        <div className="flex flex-1 w-full h-full bg-base-100 relative z-0 overflow-hidden">
            <div
                className={`${overlay ? 'overlay' : ''} h-full w-full overflow-auto`}
                id="capture-list"
            >
                <div className="w-full" id="capture-dashboard-stretch">
                    <CollapsibleSection title="Deployment Details">
                        <div className="flex mb-4 flex-wrap pdf-page">
                            <Metadata
                                className="mx-4 bg-base-100 h-48 mb-4"
                                keyValuePairs={metadataKeyValuePairs}
                                labels={labels}
                                annotations={annotations}
                            />
                            {cluster && (
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.CLUSTER}
                                    entityId={cluster.id}
                                    name="Cluster"
                                    value={cluster.name}
                                />
                            )}
                            {namespace && (
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.NAMESPACE}
                                    entityId={namespaceId}
                                    name="Namespace"
                                    value={namespace}
                                />
                            )}
                            {serviceAccount && (
                                <RelatedEntity
                                    className="mx-4 min-w-48 h-48 mb-4"
                                    entityType={entityTypes.SERVICE_ACCOUNT}
                                    name="Service Account"
                                    value={serviceAccount}
                                    entityId={serviceAccountID}
                                />
                            )}
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Images"
                                value={imageCount}
                                entityType={entityTypes.IMAGE}
                            />
                            <RelatedEntityListCount
                                className="mx-4 min-w-48 h-48 mb-4"
                                name="Secrets"
                                value={secretCount}
                                entityType={entityTypes.SECRET}
                            />
                        </div>
                    </CollapsibleSection>
                    <CollapsibleSection title="Deployment Findings">
                        <div className="flex mb-4 pdf-page pdf-stretch">
                            <p>Deployment Findings go here</p>
                        </div>
                    </CollapsibleSection>
                </div>
            </div>
        </div>
    );
    //         }}
    //     </Query>
    // );
};

VulmMgmtDeployment.propTypes = entityComponentPropTypes;
VulmMgmtDeployment.defaultProps = entityComponentDefaultProps;

export default VulmMgmtDeployment;
