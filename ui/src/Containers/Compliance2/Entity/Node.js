import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import IconWidget from 'Components/IconWidget';
import InfoWidget from 'Components/InfoWidget';
import Query from 'Components/ThrowingQuery';
import { NODE_QUERY as QUERY } from 'queries/node';
import * as Icon from 'react-feather';
import { format } from 'date-fns';
import Widget from 'Components/Widget';
import Header from './Header';

function processData(data) {
    if (!data || !data.results) return {};

    const result = { ...data.results };
    const [ipAddress] = result.internalIpAddresses;
    result.ipAddress = ipAddress;

    const joinedAt = new Date(result.joinedAt);
    result.joinedAtDate = format(joinedAt, 'MM/DD/YYYY');
    result.joinedAtTime = format(joinedAt, 'h:mm:ss:A');
    return result;
}

const NodePage = ({ sidePanelMode, params }) => (
    <Query query={QUERY} variables={{ id: params.entityId }} pollInterval={5000}>
        {({ loading, data }) => {
            const node = processData(data);
            const header = node.name || 'Loading...';
            return (
                <section className="flex flex-col h-full w-full">
                    {!sidePanelMode && (
                        <Header header={header} subHeader="Node" scanCluster={node.clusterId} />
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
                                entityType={entityTypes.NODE}
                                entityId={params.entityId}
                                entityName={node.name}
                            />
                            <div className="grid md:sx-2 md:grid-auto-fit md:grid-dense">
                                <div className="pr-3">
                                    <InfoWidget
                                        title="Operating System"
                                        headline={node.osImage}
                                        description={node.kernelVersion}
                                        loading={loading}
                                    />
                                </div>
                                <div className="pl-3">
                                    <InfoWidget
                                        title="Node Join Time"
                                        headline={node.joinedAtDate}
                                        description={node.joinedAtTime}
                                        loading={loading}
                                    />
                                </div>
                            </div>
                            <Widget header="Labels" className="s-2">
                                TBD
                            </Widget>
                            <IconWidget
                                title="Parent Cluster"
                                icon={<Icon.AlertTriangle />}
                                description={node.clusterName}
                                loading={loading}
                            />
                            <IconWidget
                                title="Container Runtime"
                                icon={<Icon.AlertTriangle />}
                                description={node.containerRuntimeVersion}
                                loading={loading}
                            />
                            <IconWidget
                                title="IP Address"
                                icon={<Icon.AlertTriangle />}
                                description={node.ipAddress}
                                loading={loading}
                            />
                            <IconWidget
                                title="Hostname"
                                icon={<Icon.AlertTriangle />}
                                description={node.name}
                                loading={loading}
                            />
                            <ComplianceByStandard type={entityTypes.PCI_DSS_3_2} params={params} />
                            <ComplianceByStandard type={entityTypes.NIST_800_190} params={params} />
                            <ComplianceByStandard type={entityTypes.HIPAA_164} params={params} />
                            <ComplianceByStandard
                                type={entityTypes.CIS_KUBERENETES_V1_2_0}
                                params={params}
                            />
                            <ComplianceByStandard
                                type={entityTypes.CIS_DOCKER_V1_1_0}
                                params={params}
                            />
                        </div>
                    </div>
                </section>
            );
        }}
    </Query>
);
NodePage.propTypes = {
    sidePanelMode: PropTypes.bool,
    params: PropTypes.shape({}).isRequired
};

NodePage.defaultProps = {
    sidePanelMode: false
};

export default NodePage;
