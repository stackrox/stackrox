import React, { useContext } from 'react';
import entityTypes from 'constants/entityTypes';
import { DEPLOYMENT_QUERY } from 'queries/deployment';
import Widget from 'Components/Widget';
import Query from 'Components/CacheFirstQuery';
import Loader from 'Components/Loader';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import ResourceTabs from 'Components/ResourceTabs';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import ComplianceList from 'Containers/Compliance/List/List';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import Cluster from 'images/cluster.svg';
import Namespace from 'images/ns-icon.svg';
import IconWidget from 'Components/IconWidget';
import pluralize from 'pluralize';
import Labels from 'Containers/Compliance/widgets/Labels';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import isGQLLoading from 'utils/gqlLoading';
import searchContext from 'Containers/searchContext';

import { knownBackendFlags } from 'utils/featureFlags';
import FeatureEnabled from 'Containers/FeatureEnabled';
import Header from './Header';

function processData(data) {
    if (!data || !data.deployment) return {};

    const result = { ...data.deployment };
    return result;
}

const DeploymentPage = ({
    match,
    location,
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
        <Query query={DEPLOYMENT_QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) return <Loader />;
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

                if (listEntityType1 && !sidePanelMode) {
                    const listQuery = {
                        groupBy:
                            listEntityType1 === entityTypes.CONTROL ? entityTypes.STANDARD : '',
                        deployment: name,
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
                    const clusterUrl = URLService.getURL(match, location)
                        .base(entityTypes.CLUSTER, clusterId)
                        .url();

                    const namespaceUrl = URLService.getURL(match, location)
                        .base(entityTypes.NAMESPACE, namespaceId)
                        .url();

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
                                <FeatureEnabled featureFlags={knownBackendFlags.ROX_NIST_800_53}>
                                    <ComplianceByStandard
                                        standardType={entityTypes.NIST_SP_800_53_Rev_4}
                                        entityName={name}
                                        entityId={id}
                                        entityType={entityTypes.DEPLOYMENT}
                                        className={pdfClassName}
                                    />
                                </FeatureEnabled>
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
                                    listEntityType={listEntityType1}
                                    entityName={name}
                                    entityId={id}
                                />
                                <ResourceTabs
                                    entityId={id}
                                    entityType={entityTypes.DEPLOYMENT}
                                    selectedType={listEntityType1}
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
DeploymentPage.propTypes = entityPagePropTypes;
DeploymentPage.defaultProps = entityPageDefaultProps;

export default withRouter(DeploymentPage);
