import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import ResourceCount from 'Containers/Compliance2/widgets/ResourceCount';
import Query from 'Components/ThrowingQuery';
import { CLUSTER_QUERY as QUERY } from 'queries/cluster';
import ResourceRelatedResourceList from 'Containers/Compliance2/widgets/ResourceRelatedResourceList';
import Header from './Header';

function processData(data) {
    if (!data || !data.results) return {};
    return data.results;
}

const ClusterPage = ({ sidePanelMode, params }) => (
    <Query query={QUERY} variables={{ id: params.entityId }} pollInterval={5000}>
        {({ loading, data }) => {
            const cluster = processData(data);
            return (
                <section className="flex flex-col h-full w-full">
                    {!sidePanelMode && (
                        <Header
                            header={cluster.name}
                            subHeader="Cluster"
                            scanCluster={params.entityId}
                            type="CLUSTER"
                        />
                    )}
                    <div
                        className={`flex-1 relative bg-base-200 overflow-auto ${
                            !sidePanelMode ? `p-6` : `p-4`
                        } `}
                    >
                        <div
                            className={`grid ${
                                !sidePanelMode
                                    ? `grid grid-gap-6 md:grid-auto-fit md:grid-dense`
                                    : ``
                            } sm:grid-columns-1 grid-gap-5`}
                        >
                            <div
                                className="grid s-2 md:grid-auto-fit md:grid-dense"
                                style={{ '--min-tile-width': '50%' }}
                            >
                                <div className="s-full pb-3">
                                    <EntityCompliance
                                        entityType={entityTypes.CLUSTER}
                                        entityId={params.entityId}
                                        entityName={cluster.name}
                                    />
                                </div>
                                <div className="md:pr-3 pt-3 rounded">
                                    <ResourceCount
                                        entityType={entityTypes.NODE}
                                        params={params}
                                        loading={loading}
                                    />
                                </div>
                                <div className="md:pl-3 pt-3 rounded">
                                    <ResourceCount
                                        entityType={entityTypes.NODE}
                                        params={params}
                                        loading={loading}
                                    />
                                </div>
                            </div>

                            <ComplianceByStandard
                                type={entityTypes.PCI_DSS_3_2}
                                entityName={cluster.name}
                                params={params}
                            />
                            <ComplianceByStandard
                                type={entityTypes.NIST_800_190}
                                entityName={cluster.name}
                                params={params}
                            />
                            <ComplianceByStandard
                                type={entityTypes.HIPAA_164}
                                entityName={cluster.name}
                                params={params}
                            />
                            <ComplianceByStandard
                                type={entityTypes.CIS_KUBERENETES_V1_2_0}
                                entityName={cluster.name}
                                params={params}
                            />
                            <ComplianceByStandard
                                type={entityTypes.CIS_DOCKER_V1_1_0}
                                entityName={cluster.name}
                                params={params}
                            />
                            {!sidePanelMode && (
                                <>
                                    <ResourceRelatedResourceList
                                        listEntityType={entityTypes.NAMESPACE}
                                        pageEntityType={entityTypes.CLUSTER}
                                        pageEntity={cluster}
                                        className="s-2"
                                    />
                                    <ResourceRelatedResourceList
                                        listEntityType={entityTypes.DEPLOYMENT}
                                        pageEntityType={entityTypes.CLUSTER}
                                        pageEntity={cluster}
                                        className="s-2"
                                    />
                                </>
                            )}
                        </div>
                    </div>
                </section>
            );
        }}
    </Query>
);

ClusterPage.propTypes = {
    sidePanelMode: PropTypes.bool,
    params: PropTypes.shape({}).isRequired
};

ClusterPage.defaultProps = {
    sidePanelMode: false
};

export default ClusterPage;
