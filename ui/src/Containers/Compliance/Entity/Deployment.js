import React from 'react';
import PropTypes from 'prop-types';
import entityTypes, { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
import { DEPLOYMENT_QUERY } from 'queries/deployment';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import ResourceTabs from 'Components/ResourceTabs';
import ComplianceList from 'Containers/Compliance/List/List';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import Cluster from 'images/cluster.svg';
import Namespace from 'images/ns-icon.svg';
import IconWidget from 'Components/IconWidget';
import pluralize from 'pluralize';
import Labels from 'Containers/Compliance/widgets/Labels';
import contextTypes from 'constants/contextTypes';

import pageTypes from 'constants/pageTypes';
import Header from './Header';
import SearchInput from '../SearchInput';

function processData(data) {
    if (!data || !data.deployment) return {};

    const result = { ...data.deployment };
    return result;
}

const DeploymentPage = ({ match, location, deploymentId, sidePanelMode }) => {
    const params = URLService.getParams(match, location);
    const entityId = deploymentId || params.entityId;
    const listEntityType = URLService.getEntityTypeKeyFromValue(params.listEntityType);

    return (
        <Query query={DEPLOYMENT_QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (loading || !data) return <Loader />;
                const deployment = processData(data);
                const {
                    name,
                    id,
                    labels,
                    clusterName,
                    namespace,
                    clusterId,
                    namespaceId
                } = deployment;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                const searchComponent = listEntityType ? (
                    <SearchInput categories={[searchCategoryTypes[listEntityType]]} />
                ) : null;

                if (listEntityType && !sidePanelMode) {
                    const queryParams = { ...params.query };
                    queryParams.deployment = name;
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
                    const clusterUrl = URLService.getLinkTo(
                        contextTypes.COMPLIANCE,
                        pageTypes.ENTITY,
                        { entityType: entityTypes.CLUSTER, entityId: clusterId }
                    );

                    const namespaceUrl = URLService.getLinkTo(
                        contextTypes.COMPLIANCE,
                        pageTypes.ENTITY,
                        { entityType: entityTypes.NAMESPACE, entityId: namespaceId }
                    );

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
                                            entityType={entityTypes.DEPLOYMENT}
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
                                            linkUrl={clusterUrl}
                                        />
                                    </div>
                                    <div className="md:pl-3 pt-3">
                                        <IconWidget
                                            title="Parent Namespace"
                                            icon={Namespace}
                                            description={namespace}
                                            loading={loading}
                                            linkUrl={namespaceUrl}
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
                                    entityType={entityTypes.DEPLOYMENT}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.NIST_800_190}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.DEPLOYMENT}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.HIPAA_164}
                                    entityName={name}
                                    entityId={id}
                                    entityType={entityTypes.DEPLOYMENT}
                                    className={pdfClassName}
                                />
                            </div>
                        </div>
                    );
                }

                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && (
                            <>
                                <Header
                                    entityType={entityTypes.DEPLOYMENT}
                                    listEntityType={listEntityType}
                                    entityName={name}
                                    entityId={id}
                                />
                                <ResourceTabs
                                    entityId={id}
                                    entityType={entityTypes.DEPLOYMENT}
                                    resourceTabs={[entityTypes.CONTROL]}
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
DeploymentPage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    deploymentId: PropTypes.string,
    sidePanelMode: PropTypes.bool
};

DeploymentPage.defaultProps = {
    deploymentId: null,
    sidePanelMode: false
};

export default withRouter(DeploymentPage);
