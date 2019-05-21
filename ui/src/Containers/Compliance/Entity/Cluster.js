import React, { useState } from 'react';
import PropTypes from 'prop-types';
import entityTypes, { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
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
import PageNotFound from 'Components/PageNotFound';
import Header from './Header';
import SearchInput from '../SearchInput';

const types = ['nodes', 'namespaces', 'deployments', 'controls'];

function processData(data) {
    if (!data || !data.results) return {};
    return data.results;
}

const ClusterPage = ({ match, location, clusterId, sidePanelMode }) => {
    const params = URLService.getParams(match, location);

    const entityId = clusterId || params.entityId;
    const listEntityType = URLService.getEntityTypeKeyFromValue(params.listEntityType);
    const [listType, setListType] = useState(listEntityType);

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data.results) return <PageNotFound resourceType={entityTypes.CLUSTER} />;
                const cluster = processData(data);
                const { name, id } = cluster;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                const searchComponent = listEntityType ? (
                    <SearchInput categories={[searchCategoryTypes[listEntityType]]} />
                ) : null;

                if (listEntityType && !sidePanelMode) {
                    const queryParams = { ...params.query };
                    queryParams['Cluster Id'] = entityId;
                    const listQuery = {
                        groupBy: listEntityType === entityTypes.CONTROL ? entityTypes.STANDARD : '',
                        ...queryParams
                    };
                    contents = (
                        <section
                            id="capture-list"
                            className="flex flex-col flex-1 overflow-y-auto h-full bg-base-100"
                        >
                            <ComplianceList
                                searchComponent={searchComponent}
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
                                <ComplianceByStandard
                                    standardType={entityTypes.HIPAA_164}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.CIS_Kubernetes_v1_2_0}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.CLUSTER}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.CIS_Docker_v1_1_0}
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

                function modifyListType(type) {
                    if (types.includes(type)) {
                        setListType(type);
                    } else {
                        setListType(null);
                    }
                }

                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && (
                            <>
                                <Header
                                    entityType={entityTypes.CLUSTER}
                                    listEntityType={listType}
                                    entityName={name}
                                    entityId={id}
                                />
                                <ResourceTabs
                                    entityId={entityId}
                                    entityType={entityTypes.CLUSTER}
                                    resourceTabs={[
                                        entityTypes.CONTROL,
                                        entityTypes.NAMESPACE,
                                        entityTypes.NODE,
                                        entityTypes.DEPLOYMENT
                                    ]}
                                    onClick={modifyListType}
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
