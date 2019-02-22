import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import { NODE_QUERY } from 'queries/node';
import { format } from 'date-fns';
import pluralize from 'pluralize';

import Cluster from 'images/cluster.svg';
import IpAddress from 'images/ip-address.svg';
import Hostname from 'images/hostname.svg';
import ContainerRuntime from 'images/container-runtime.svg';

import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import IconWidget from 'Components/IconWidget';
import InfoWidget from 'Components/InfoWidget';
import Labels from 'Containers/Compliance/widgets/Labels';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import ComplianceByStandard from 'Containers/Compliance/widgets/ComplianceByStandard';
import Loader from 'Components/Loader';
import Header from './Header';

function processData(data) {
    if (!data || !data.node) return {};

    const result = { ...data.node };
    const [ipAddress] = result.internalIpAddresses;
    result.ipAddress = ipAddress;

    const joinedAt = new Date(result.joinedAt);
    result.joinedAtDate = format(joinedAt, 'MM/DD/YYYY');
    result.joinedAtTime = format(joinedAt, 'h:mm:ss:A');
    return result;
}

const NodePage = ({ sidePanelMode, params }) => (
    <Query query={NODE_QUERY} variables={{ id: params.entityId }}>
        {({ loading, data }) => {
            if (loading || !data) return <Loader />;
            const node = processData(data);
            const header = node.name || 'Loading...';
            const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
            return (
                <section className="flex flex-col h-full w-full">
                    {!sidePanelMode && <Header header={header} subHeader="Node" params={params} />}
                    <div
                        className={`flex-1 relative bg-base-200 overflow-auto ${
                            !sidePanelMode ? `p-6` : `p-4`
                        } `}
                        id="capture-dashboard"
                    >
                        <div
                            style={{ '--min-tile-height': '190px' }}
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
                                        entityType={entityTypes.NODE}
                                        entityId={params.entityId}
                                        entityName={node.name}
                                        clusterName={node.clusterName}
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
                                className={`grid s-2 md:grid-auto-fit md:grid-dense ${pdfClassName}`}
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

                            <Widget
                                className={`sx-2 ${pdfClassName}`}
                                header={`${node.labels.length} ${pluralize(
                                    'Label',
                                    node.labels.length
                                )}`}
                            >
                                <Labels labels={node.labels} />
                            </Widget>
                            <ComplianceByStandard
                                type={entityTypes.NIST_800_190}
                                entityName={node.name}
                                params={params}
                                className={pdfClassName}
                            />
                            <ComplianceByStandard
                                type={entityTypes.CIS_Kubernetes_v1_2_0}
                                entityName={node.name}
                                params={params}
                                className={pdfClassName}
                            />
                            <ComplianceByStandard
                                type={entityTypes.CIS_Docker_v1_1_0}
                                entityName={node.name}
                                params={params}
                                className={pdfClassName}
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
