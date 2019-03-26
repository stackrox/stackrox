import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import ResourceCount from 'Containers/Compliance/widgets/ResourceCount';
import ClusterVersion from 'Containers/Compliance/widgets/ClusterVersion';
import Query from 'Components/ThrowingQuery';
import { CLUSTER_QUERY as QUERY } from 'queries/cluster';
import ComplianceList from 'Containers/Compliance/List/List';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import Loader from 'Components/Loader';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import ResourceTabs from 'Components/ResourceTabs';
import Header from './Header';

function processData(data) {
    if (!data || !data.results) return {};
    return data.results;
}

const ClusterPage = ({ match, location, clusterId, sidePanelMode }) => {
    const params = URLService.getParams(match, location);
    const entityId = clusterId || params.entityId;
    const listEntityType = URLService.getEntityTypeKeyFromValue(params.listEntityType);

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                const cluster = processData(data);
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;
                if (listEntityType && !sidePanelMode) {
                    const listQuery = {
                        'Cluster Id': entityId,
                        groupBy: listEntityType === entityTypes.CONTROL ? entityTypes.STANDARD : ''
                    };
                    contents = (
                        <section id="capture-list">
                            <ComplianceList
                                entityType={listEntityType}
                                query={listQuery}
                                className={pdfClassName}
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
                                            entityType={entityTypes.CLUSTER}
                                            entityId={cluster.id}
                                            entityName={cluster.name}
                                            clusterName={cluster.name}
                                        />
                                    </div>
                                    <div className="s-full">
                                        <ClusterVersion clusterId={cluster.id} />
                                    </div>
                                </div>
                                <ComplianceByStandard
                                    standardType={entityTypes.PCI_DSS_3_2}
                                    entityName={cluster.name}
                                    entityId={cluster.id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.NIST_800_190}
                                    entityName={cluster.name}
                                    entityId={cluster.id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.HIPAA_164}
                                    entityName={cluster.name}
                                    entityId={cluster.id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.CIS_Kubernetes_v1_2_0}
                                    entityName={cluster.name}
                                    entityId={cluster.id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.CIS_Docker_v1_1_0}
                                    entityName={cluster.name}
                                    entityId={cluster.id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                {sidePanelMode && (
                                    <>
                                        <div
                                            className={`grid s-2 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
                                            style={{ '--min-tile-width': '50%' }}
                                        >
                                            <div className="md:pr-3 pb-3">
                                                <ResourceCount
                                                    entityType={entityTypes.NAMESPACE}
                                                    relatedToResourceType={entityTypes.CLUSTER}
                                                    relatedToResource={cluster}
                                                />
                                            </div>
                                            <div className="md:pl-3 pb-3">
                                                <ResourceCount
                                                    entityType={entityTypes.NODE}
                                                    relatedToResourceType={entityTypes.CLUSTER}
                                                    relatedToResource={cluster}
                                                />
                                            </div>
                                            <div className="md:pr-3 pt-2">
                                                <ResourceCount
                                                    entityType={entityTypes.DEPLOYMENT}
                                                    relatedToResourceType={entityTypes.CLUSTER}
                                                    relatedToResource={cluster}
                                                />
                                            </div>
                                        </div>
                                    </>
                                )}
                            </div>
                        </div>
                    );
                }
                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && (
                            <>
                                <Header
                                    entityType={entityTypes.CLUSTER}
                                    listEntityType={listEntityType}
                                    entity={cluster}
                                />
                                <ResourceTabs
                                    entityId={entityId}
                                    entityType={entityTypes.CLUSTER}
                                    resourceTabs={[
                                        entityTypes.CONTROL,
                                        entityTypes.NAMESPACE,
                                        entityTypes.NODE
                                    ]}
                                />
                            </>
                        )}

                        {contents}
                    </section>
                );
            }}
        </Query>
    );
};

ClusterPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    clusterId: PropTypes.string,
    sidePanelMode: PropTypes.bool
};

ClusterPage.defaultProps = {
    clusterId: null,
    sidePanelMode: false
};

export default withRouter(ClusterPage);
