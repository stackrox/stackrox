import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import ResourceCount from 'Containers/Compliance/widgets/ResourceCount';
import ClusterVersion from 'Containers/Compliance/widgets/ClusterVersion';
import Query from 'Components/CacheFirstQuery';
import { CLUSTER_QUERY as QUERY } from 'queries/cluster';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import ComplianceList from 'Containers/Compliance/List/List';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import Loader from 'Components/Loader';
import { withRouter } from 'react-router-dom';
import ResourceTabs from 'Components/ResourceTabs';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import searchContext from 'Containers/searchContext';
import isGQLLoading from 'utils/gqlLoading';
import FeatureEnabled from 'Containers/FeatureEnabled';
import { knownBackendFlags } from 'utils/featureFlags';
import Header from './Header';

function processData(data) {
    if (!data || !data.results) return {};
    return data.results;
}

const ClusterPage = ({
    entityId,
    listEntityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query,
    sidePanelMode
}) => {
    const searchParam = useContext(searchContext);

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) return <Loader transparent />;
                const cluster = processData(data);
                const { name, id } = cluster;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                if (listEntityType1 && !sidePanelMode) {
                    const listQuery = {
                        groupBy:
                            listEntityType1 === entityTypes.CONTROL ? entityTypes.STANDARD : '',
                        'Cluster Id': entityId,
                        ...query[searchParam]
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
                                            entityType={entityTypes.CLUSTER}
                                            entityId={id}
                                            entityName={name}
                                            clusterName={name}
                                        />
                                    </div>
                                    <div className="s-full">
                                        <ClusterVersion clusterId={id} />
                                    </div>
                                </div>
                                <ComplianceByStandard
                                    standardType={entityTypes.PCI_DSS_3_2}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.NIST_800_190}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <FeatureEnabled featureFlag={knownBackendFlags.ROX_NIST_800_53}>
                                    <ComplianceByStandard
                                        standardType={entityTypes.NIST_SP_800_53}
                                        entityName={name}
                                        entityId={id}
                                        entityType={entityTypes.CLUSTER}
                                        className={pdfClassName}
                                    />
                                </FeatureEnabled>
                                <ComplianceByStandard
                                    standardType={entityTypes.HIPAA_164}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.CIS_Kubernetes_v1_5}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.CIS_Docker_v1_2_0}
                                    entityName={name}
                                    entityId={id}
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
                                    listEntityType={listEntityType1}
                                    entityName={name}
                                    entityId={id}
                                />
                                <ResourceTabs
                                    entityId={entityId}
                                    entityType={entityTypes.CLUSTER}
                                    selectedType={listEntityType1}
                                    resourceTabs={[
                                        entityTypes.CONTROL,
                                        entityTypes.NAMESPACE,
                                        entityTypes.NODE,
                                        entityTypes.DEPLOYMENT
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

ClusterPage.propTypes = entityPagePropTypes;
ClusterPage.defaultProps = entityPageDefaultProps;

export default withRouter(ClusterPage);
