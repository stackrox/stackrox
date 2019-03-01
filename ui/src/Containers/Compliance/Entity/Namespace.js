import React from 'react';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import Query from 'Components/ThrowingQuery';
import Labels from 'Containers/Compliance/widgets/Labels';
import IconWidget from 'Components/IconWidget';
import CountWidget from 'Components/CountWidget';
import pluralize from 'pluralize';
import Cluster from 'images/cluster.svg';
import { NAMESPACE_QUERY as QUERY } from 'queries/namespace';
import Widget from 'Components/Widget';
import ResourceRelatedResourceList from 'Containers/Compliance/widgets/ResourceRelatedResourceList';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import Header from './Header';

function processData(data) {
    const defaultValue = {
        labels: [],
        name: '',
        clusterName: ''
    };

    if (!data || !data.results || !data.results.metadata) return defaultValue;

    const { metadata, ...rest } = data.results;

    return {
        ...rest,
        ...metadata
    };
}

const NamespacePage = ({ match, location, namespaceId, sidePanelMode }) => {
    const params = URLService.getParams(match, location);
    const entityId = namespaceId || params.entityId;

    return (
        <Query query={QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                const namespace = processData(data);
                const header = namespace.name || 'Loading';
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                return (
                    <section className="flex flex-col h-full w-full">
                        {!sidePanelMode && <Header header={header} subHeader="Namespace" />}
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
                                            entityId={namespace.id}
                                            entityName={namespace.name}
                                            clusterName={namespace.clusterName}
                                        />
                                    </div>
                                    <div className="md:pr-3 pt-3">
                                        <IconWidget
                                            title="Parent Cluster"
                                            icon={Cluster}
                                            description={namespace.clusterName}
                                            loading={loading}
                                        />
                                    </div>
                                    <div className="md:pl-3 pt-3">
                                        <CountWidget
                                            title="Network Policies"
                                            count={namespace.numNetworkPolicies}
                                        />
                                    </div>
                                </div>

                                <Widget
                                    className={`sx-2 ${pdfClassName}`}
                                    header={`${namespace.labels.length} ${pluralize(
                                        'Label',
                                        namespace.labels.length
                                    )}`}
                                >
                                    <Labels labels={namespace.labels} />
                                </Widget>

                                <ComplianceByStandard
                                    standardType={entityTypes.PCI_DSS_3_2}
                                    entityName={namespace.name}
                                    entityId={namespace.id}
                                    entityType={entityTypes.NAMESPACE}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.NIST_800_190}
                                    entityName={namespace.name}
                                    entityId={namespace.id}
                                    entityType={entityTypes.NAMESPACE}
                                    className={pdfClassName}
                                />
                                <ComplianceByStandard
                                    standardType={entityTypes.HIPAA_164}
                                    entityName={namespace.name}
                                    entityId={namespace.id}
                                    entityType={entityTypes.NAMESPACE}
                                    className={pdfClassName}
                                />
                                {!sidePanelMode && (
                                    <>
                                        <ResourceRelatedResourceList
                                            listEntityType={entityTypes.DEPLOYMENT}
                                            pageEntityType={entityTypes.NAMESPACE}
                                            pageEntity={namespace}
                                            clusterName={namespace.clusterName}
                                            className={`sx-2 ${pdfClassName}`}
                                        />
                                        <ResourceRelatedResourceList
                                            listEntityType={entityTypes.SECRET}
                                            pageEntityType={entityTypes.NAMESPACE}
                                            pageEntity={namespace}
                                            clusterName={namespace.clusterName}
                                            className={`sx-2 ${pdfClassName}`}
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
