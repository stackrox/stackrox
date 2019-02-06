import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import Query from 'Components/ThrowingQuery';
import IconWidget from 'Components/IconWidget';
import CountWidget from 'Components/CountWidget';
import * as Icon from 'react-feather';
import { NAMESPACE_QUERY as QUERY } from 'queries/namespace';
import Widget from 'Components/Widget';
import Header from './Header';

function processData(data) {
    const defaultValue = {
        labels: []
    };
    if (!data || !data.results || !data.results.metadata) return defaultValue;

    const { metadata, ...rest } = data.results;

    return {
        ...rest,
        ...metadata
    };
}

const NamespacePage = ({ sidePanelMode, params }) => (
    <Query query={QUERY} variables={{ id: params.entityId }} pollInterval={5000}>
        {({ loading, data }) => {
            const namespace = processData(data);
            const header = namespace.name || 'Loading';
            return (
                <section className="flex flex-col h-full w-full">
                    {!sidePanelMode && <Header header={header} subHeader="Namespace" />}
                    <div className="flex-1 relative bg-base-200 p-6 overflow-auto">
                        <div
                            className={`grid ${
                                !sidePanelMode
                                    ? `grid grid-gap-6 md:grid-auto-fit md:grid-dense`
                                    : ``
                            } sm:grid-columns-1 grid-gap-6`}
                        >
                            <EntityCompliance
                                entityType={entityTypes.NAMESPACE}
                                entityId={params.entityId}
                                entityName={namespace.name}
                            />
                            <Widget header={`${namespace.labels.length} Labels`}>
                                <ul>
                                    {namespace.labels.map(label => (
                                        <li key={label.value}>{label.value}</li>
                                    ))}
                                </ul>
                            </Widget>
                            <ComplianceByStandard type={entityTypes.PCI_DSS_3_2} params={params} />
                            <ComplianceByStandard type={entityTypes.NIST_800_190} params={params} />
                            <div className="grid md:sx-2 md:grid-auto-fit md:grid-dense">
                                <div className="pr-3">
                                    <IconWidget
                                        title="Parent Cluster"
                                        icon={<Icon.AlertTriangle />}
                                        description={namespace.clusterName}
                                        loading={loading}
                                    />
                                </div>
                                <div className="pl-3">
                                    <CountWidget
                                        title="Network Policies"
                                        count={namespace.numNetworkPolicies}
                                    />
                                    ;
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
                            {/* {!sidePanelMode && (
                                <RelatedEntitiesList
                                    type={entityTypes.DEPLOYMENT}
                                    params={params}
                                />
                            )} */}
                        </div>
                    </div>
                </section>
            );
        }}
    </Query>
);

NamespacePage.propTypes = {
    sidePanelMode: PropTypes.bool,
    params: PropTypes.shape({}).isRequired
};

NamespacePage.defaultProps = {
    sidePanelMode: false
};

export default NamespacePage;
