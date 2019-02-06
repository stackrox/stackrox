import React from 'react';
import ComplianceByStandard from 'Containers/Compliance2/widgets/ComplianceByStandard';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import EntityCompliance from 'Containers/Compliance2/widgets/EntityCompliance';
import Labels from 'Containers/Compliance2/widgets/Labels';
import IconWidget from 'Components/IconWidget';
import InfoWidget from 'Components/InfoWidget';
import Query from 'Components/ThrowingQuery';
import { NODE_QUERY as QUERY } from 'queries/node';
import { format } from 'date-fns';
import Widget from 'Components/Widget';
import Cluster from 'images/cluster.svg';
import IpAddress from 'images/ip-address.svg';
import Hostname from 'images/hostname.svg';
import ContainerRuntime from 'images/container-runtime.svg';
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

const labels = ['label1 has long text to show how this works', 'label2', 'label3', 'label4'];
const manyLabels = [
    'label1 has long text to show how this works',
    'label2',
    'label3',
    'label4',
    'label5',
    'label6',
    'label7',
    'label8',
    'label9',
    'label10',
    'label11',
    'label12'
];

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
                            // because of the lack of responsive bar chart
                            style={{ '--min-tile-height': '190px' }}
                            className={`grid ${
                                !sidePanelMode ? `grid-gap-6 md:grid-auto-fit md:grid-dense` : ``
                            } sm:grid-columns-1 grid-gap-6`}
                        >
                            <div
                                className="grid s-2 md:grid-auto-fit md:grid-dense"
                                style={{ '--min-tile-width': '50%' }}
                            >
                                <div className="s-full pb-3">
                                    <EntityCompliance
                                        entityType={entityTypes.NODE}
                                        entityId={params.entityId}
                                        entityName={node.name}
                                    />
                                </div>
                                <div className="md:pr-3 pt-3">
                                    <IconWidget
                                        title="Parent Cluster"
                                        icon={Cluster}
                                        description={node.clusterName}
                                        loading={loading}
                                    />
                                </div>
                                <div className="md:pl-3 pt-3">
                                    <IconWidget
                                        title="Container Runtime"
                                        icon={ContainerRuntime}
                                        description={node.containerRuntimeVersion}
                                        loading={loading}
                                    />
                                </div>
                            </div>

                            <div
                                className="grid s-2 md:grid-auto-fit md:grid-dense"
                                style={{ '--min-tile-width': '50%' }}
                            >
                                <div className="md:pr-3 pb-3">
                                    <InfoWidget
                                        title="Operating System"
                                        headline={node.osImage}
                                        description={node.kernelVersion}
                                        loading={loading}
                                    />
                                </div>
                                <div className="md:pl-3 pb-3">
                                    <InfoWidget
                                        title="Node Join Time"
                                        headline={node.joinedAtDate}
                                        description={node.joinedAtTime}
                                        loading={loading}
                                    />
                                </div>
                                <div className="md:pr-3 pt-3">
                                    <IconWidget
                                        title="IP Address"
                                        icon={IpAddress}
                                        description={node.ipAddress}
                                        loading={loading}
                                    />
                                </div>
                                <div className="md:pl-3 pt-3">
                                    <IconWidget
                                        title="Hostname"
                                        icon={Hostname}
                                        textSizeClass="text-base"
                                        description={node.name}
                                        loading={loading}
                                    />
                                </div>
                            </div>

                            <Widget className="sx-2" header="labels here">
                                <Labels list={labels} />
                            </Widget>
                            <Widget className="sx-2" header="labels here">
                                <Labels list={manyLabels} />
                            </Widget>

                            <Widget header="Annotations" className="sx-2">
                                <div className="p-3 overflow-auto leading-loose">
                                    <p>
                                        The metadata in an annotation can be small or large,
                                        structured or unstructured, but Gorman doesn’t see this
                                        becoming too large. I think given the nature of the content,
                                        we should let the widget grow to match it. If there are
                                        special cases where this is incredibly long, we can consider
                                        introducting a max-height boundary and enabling overflow
                                        (though less ideal)
                                    </p>
                                </div>
                            </Widget>

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
