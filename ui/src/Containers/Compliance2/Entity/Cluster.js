import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import RelatedEntitiesList from 'Containers/Compliance2/widgets/RelatedEntitiesList';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import ResourceCount from 'Containers/Compliance2/widgets/ResourceCount';
import Query from 'Components/ThrowingQuery';
import { CLUSTER_QUERY as QUERY } from 'queries/cluster';
import Header from './Header';

function processData(data) {
    if (!data || !data.results) return {};
    return data.results;
}

const ClusterPage = ({ sidePanelMode, params }) => (
    <Query query={QUERY} variables={{ id: params.entityId }} pollInterval={5000}>
        {({ loading, data }) => {
            const cluster = processData(data);
            const header = cluster.name;
            return (
                <section className="flex flex-col h-full w-full">
                    {!sidePanelMode && (
                        <Header
                            header={header}
                            subHeader="Cluster"
                            scanCluster={params.entityId}
                            type="CLUSTER"
                        />
                    )}
                    <div className="flex-1 relative bg-base-200 p-6 overflow-auto">
                        <div
                            className={`grid ${
                                !sidePanelMode
                                    ? `grid grid-gap-6 md:grid-auto-fit md:grid-dense`
                                    : ``
                            } sm:grid-columns-1 grid-gap-6`}
                        >
                            <EntityCompliance
                                entityType={entityTypes.CLUSTER}
                                entityId={params.entityId}
                                entityName={cluster.name}
                            />

                            <ComplianceByStandard type={entityTypes.PCI_DSS_3_2} params={params} />
                            <ComplianceByStandard type={entityTypes.NIST_800_190} params={params} />
                            <div className="grid md:sx-2 md:grid-auto-fit md:grid-dense">
                                <div className="pr-3">
                                    <ResourceCount
                                        entityType={entityTypes.NODE}
                                        params={params}
                                        loading={loading}
                                    />
                                </div>
                                <div className="pl-3">
                                    <ResourceCount
                                        entityType={entityTypes.NODE}
                                        params={params}
                                        loading={loading}
                                    />
                                </div>
                            </div>

                            <ComplianceByStandard type={entityTypes.HIPAA_164} params={params} />
                            <ComplianceByStandard
                                type={entityTypes.CIS_KUBERENETES_V1_2_0}
                                params={params}
                            />
                            <ComplianceByStandard
                                type={entityTypes.CIS_DOCKER_V1_1_0}
                                params={params}
                            />
                            {!sidePanelMode && (
                                <RelatedEntitiesList
                                    type={entityTypes.DEPLOYMENT}
                                    params={params}
                                />
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
