import React, { useContext, useState } from 'react';

import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import ResourceCount from 'Containers/Compliance/widgets/ResourceCount';
import ClusterVersion from 'Containers/Compliance/widgets/ClusterVersion';
import Query from 'Components/CacheFirstQuery';
import usePermissions from 'hooks/usePermissions';
import { CLUSTER_NAME as QUERY } from 'queries/cluster';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable import/no-cycle */
import ComplianceList from 'Containers/Compliance/List/List';
import ComplianceByStandards from 'Containers/Compliance/widgets/ComplianceByStandards';
import Loader from 'Components/Loader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import isGQLLoading from 'utils/gqlLoading';

import EntityHeader from './EntityHeader';
import ResourceTabs from './ResourceTabs';
import { isComplianceRouteEnabled } from '../complianceRBAC';

function getResourceTabs({
    isComplianceRouteEnabledForDeployments,
    isComplianceRouteEnabledForNamespaces,
    isComplianceRouteEnabledForNodes,
}) {
    const resourceTabs = ['CONTROL'];
    if (isComplianceRouteEnabledForNamespaces) {
        resourceTabs.push('NAMESPACE');
    }
    if (isComplianceRouteEnabledForNodes) {
        resourceTabs.push('NODE');
    }
    if (isComplianceRouteEnabledForDeployments) {
        resourceTabs.push('DEPLOYMENT');
    }
    return resourceTabs;
}

function processData(data) {
    if (!data || !data.cluster) {
        return {};
    }
    return data.cluster;
}

const ComplianceEntityCluster = ({
    entityId,
    listEntityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query,
    sidePanelMode,
}) => {
    const [isExporting, setIsExporting] = useState(false);
    const searchParam = useContext(searchContext);

    const { hasReadAccess } = usePermissions();
    const isComplianceRouteEnabledForDeployments = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/deployments'
    );
    const isComplianceRouteEnabledForNamespaces = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/namespaces'
    );
    const isComplianceRouteEnabledForNodes = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/nodes'
    );
    const isComplianceRouteEnabledForEntities =
        isComplianceRouteEnabledForDeployments ||
        isComplianceRouteEnabledForNamespaces ||
        isComplianceRouteEnabledForNodes;

    const resourceTabs = getResourceTabs({
        isComplianceRouteEnabledForDeployments,
        isComplianceRouteEnabledForNamespaces,
        isComplianceRouteEnabledForNodes,
    });

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }
                const cluster = processData(data);
                const { name, id } = cluster;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                if (listEntityType1 && !sidePanelMode) {
                    const listQuery = {
                        groupBy: listEntityType1 === 'CONTROL' ? 'STANDARD' : '',
                        'Cluster Id': entityId,
                        ...query[searchParam],
                    };
                    contents = (
                        <section
                            id="capture-list"
                            className="flex flex-col flex-1 overflow-y-auto h-full"
                        >
                            <ComplianceList
                                entityType={listEntityType1}
                                query={listQuery}
                                selectedRowId={entityId1}
                                entityType2={entityType2}
                                entityListType2={entityListType2}
                                entityId2={entityId2}
                            />
                        </section>
                    );
                } else {
                    contents = (
                        <div
                            className={`flex-1 relative bg-base-200 overflow-auto ${
                                !sidePanelMode ? `p-6` : `p-4`
                            } `}
                            id="capture-dashboard"
                        >
                            <div
                                className={`grid ${
                                    !sidePanelMode
                                        ? `grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense`
                                        : ``
                                } sm:grid-columns-1 grid-gap-5`}
                            >
                                <div
                                    className={`grid s-2 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
                                    style={{ '--min-tile-width': '50%' }}
                                >
                                    <div className="s-full pb-5">
                                        <EntityCompliance
                                            entityType="CLUSTER"
                                            entityId={id}
                                            entityName={name}
                                            clusterName={name}
                                        />
                                    </div>
                                    <div className="s-full">
                                        <ClusterVersion clusterId={id} />
                                    </div>
                                </div>
                                <ComplianceByStandards
                                    entityId={id}
                                    entityName={name}
                                    entityType="CLUSTER"
                                />

                                {sidePanelMode && isComplianceRouteEnabledForEntities && (
                                    <div
                                        className={`grid s-2 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
                                        style={{ '--min-tile-width': '50%' }}
                                    >
                                        {isComplianceRouteEnabledForNamespaces && (
                                            <div className="md:pr-3 pb-3">
                                                <ResourceCount
                                                    entityType="NAMESPACE"
                                                    relatedToResourceType="CLUSTER"
                                                    relatedToResource={cluster}
                                                />
                                            </div>
                                        )}
                                        {isComplianceRouteEnabledForNodes && (
                                            <div className="md:pl-3 pb-3">
                                                <ResourceCount
                                                    entityType="NODE"
                                                    relatedToResourceType="CLUSTER"
                                                    relatedToResource={cluster}
                                                />
                                            </div>
                                        )}
                                        {isComplianceRouteEnabledForDeployments && (
                                            <div className="md:pr-3 pt-2">
                                                <ResourceCount
                                                    entityType="DEPLOYMENT"
                                                    relatedToResourceType="CLUSTER"
                                                    relatedToResource={cluster}
                                                />
                                            </div>
                                        )}
                                    </div>
                                )}
                            </div>
                        </div>
                    );
                }

                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && (
                            <>
                                <EntityHeader
                                    entityType="CLUSTER"
                                    listEntityType={listEntityType1}
                                    entityName={name}
                                    entityId={id}
                                    isExporting={isExporting}
                                    setIsExporting={setIsExporting}
                                />
                                <ResourceTabs
                                    entityId={entityId}
                                    entityType="CLUSTER"
                                    selectedType={listEntityType1}
                                    resourceTabs={resourceTabs}
                                />
                            </>
                        )}
                        {contents}
                        {isExporting && <BackdropExporting />}
                    </section>
                );
            }}
        </Query>
    );
};

ComplianceEntityCluster.propTypes = entityPagePropTypes;
ComplianceEntityCluster.defaultProps = entityPageDefaultProps;

export default ComplianceEntityCluster;
