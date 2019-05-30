import React from 'react';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import PropTypes from 'prop-types';
import entityTypes, { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import Query from 'Components/ThrowingQuery';
import Labels from 'Containers/Compliance/widgets/Labels';
import IconWidget from 'Components/IconWidget';
import CountWidget from 'Components/CountWidget';
import pluralize from 'pluralize';
import Cluster from 'images/cluster.svg';
import { NAMESPACE_QUERY as QUERY } from 'queries/namespace';
import Widget from 'Components/Widget';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import ComplianceList from 'Containers/Compliance/List/List';
import ResourceTabs from 'Components/ResourceTabs';
import ResourceCount from 'Containers/Compliance/widgets/ResourceCount';
import PageNotFound from 'Components/PageNotFound';
import Loader from 'Components/Loader';
import Header from './Header';
import SearchInput from '../SearchInput';

const NamespacePage = ({ match, location, namespaceId, sidePanelMode }) => {
    const params = URLService.getParams(match, location);
    const entityId = namespaceId || params.entityId;
    const listEntityType = URLService.getEntityTypeKeyFromValue(params.listEntityType);

    function processData(data) {
        const defaultValue = {
            labels: [],
            name: '',
            clusterName: '',
            id: entityId
        };

        if (!data || !data.results || !data.results.metadata) return defaultValue;

        const { metadata, ...rest } = data.results;

        return {
            ...rest,
            ...metadata
        };
    }

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data.results) return <PageNotFound resourceType={entityTypes.NAMESPACE} />;
                const namespace = processData(data);
                const { name, id, clusterName, labels, numNetworkPolicies } = namespace;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                const searchComponent = listEntityType ? (
                    <SearchInput categories={[searchCategoryTypes[listEntityType]]} />
                ) : null;

                if (listEntityType && !sidePanelMode) {
                    const queryParams = { ...params.query };
                    queryParams.Namespace = namespace.name;
                    const listQuery = {
                        groupBy: listEntityType === entityTypes.CONTROL ? entityTypes.STANDARD : '',
                        ...queryParams
                    };
                    contents = (
                        <section
                            id="capture-list"
                            className="flex flex-col flex-1 overflow-y-auto h-full"
                        >
                            <ComplianceList
                                searchComponent={searchComponent}
                                entityType={listEntityType}
                                query={listQuery}
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
                                    <div className="s-full pb-3">
                                        <EntityCompliance
                                            entityType={entityTypes.NAMESPACE}
                                            entityId={id}
                                            entityName={name}
                                            clusterName={clusterName}
                                        />
                                    </div>
                                    <div className="md:pr-3 pt-3">
                                        <IconWidget
                                            title="Parent Cluster"
                                            icon={Cluster}
                                            description={clusterName}
                                            loading={loading}
                                        />
                                    </div>
                                    <div className="md:pl-3 pt-3">
                                        <CountWidget
                                            title="Network Policies"
                                            count={numNetworkPolicies}
                                        />
                                    </div>
                                </div>

                                <Widget
                                    className={`sx-2 ${pdfClassName}`}
                                    header={`${labels.length} ${pluralize('Label', labels.length)}`}
                                >
                                    <Labels labels={labels} />
                                </Widget>

                                <ComplianceByStandard
                                    standardType={entityTypes.PCI_DSS_3_2}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.NAMESPACE}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.NIST_800_190}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.NAMESPACE}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.HIPAA_164}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.NAMESPACE}
                                    className={pdfClassName}
                                />
                                {sidePanelMode && (
                                    <>
                                        <div
                                            className={`grid sx-2 sy-1 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
                                            style={{ '--min-tile-width': '50%' }}
                                        >
                                            <div className="md:pr-3 pt-3">
                                                <ResourceCount
                                                    entityType={entityTypes.DEPLOYMENT}
                                                    relatedToResourceType={entityTypes.NAMESPACE}
                                                    relatedToResource={namespace}
                                                />
                                            </div>
                                            <div className="md:pl-3 pt-3">
                                                <ResourceCount
                                                    entityType={entityTypes.SECRET}
                                                    relatedToResourceType={entityTypes.NAMESPACE}
                                                    relatedToResource={namespace}
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
                                    entityType={entityTypes.NAMESPACE}
                                    listEntityType={listEntityType}
                                    entityName={name}
                                    entityId={id}
                                />
                                <ResourceTabs
                                    entityId={id}
                                    entityType={entityTypes.NAMESPACE}
                                    resourceTabs={[entityTypes.CONTROL, entityTypes.DEPLOYMENT]}
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

NamespacePage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    namespaceId: PropTypes.string,
    sidePanelMode: PropTypes.bool
};

NamespacePage.defaultProps = {
    namespaceId: null,
    sidePanelMode: false
};

export default withRouter(NamespacePage);
