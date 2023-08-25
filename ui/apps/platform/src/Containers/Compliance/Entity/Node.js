import React, { useContext, useState } from 'react';
import { format } from 'date-fns';
import pluralize from 'pluralize';

import entityTypes from 'constants/entityTypes';
import { NODE_QUERY } from 'queries/node';
import Cluster from 'images/cluster.svg';
import IpAddress from 'images/ip-address.svg';
import Hostname from 'images/hostname.svg';
import ContainerRuntime from 'images/container-runtime.svg';
import ComplianceByStandards from 'Containers/Compliance/widgets/ComplianceByStandards';
import Widget from 'Components/Widget';
import Query from 'Components/CacheFirstQuery';
import IconWidget from 'Components/IconWidget';
import InfoWidget from 'Components/InfoWidget';
import Labels from 'Containers/Compliance/widgets/Labels';
import EntityCompliance from 'Containers/Compliance/widgets/EntityCompliance';
import Loader from 'Components/Loader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
// TODO: this exception will be unnecessary once Compliance pages are re-structured like Config Management
/* eslint-disable-next-line import/no-cycle */
import ComplianceList from 'Containers/Compliance/List/List';
import PageNotFound from 'Components/PageNotFound';
import { entityPagePropTypes, entityPageDefaultProps } from 'constants/entityPageProps';
import useCases from 'constants/useCaseTypes';
import isGQLLoading from 'utils/gqlLoading';
import searchContext from 'Containers/searchContext';

import Header from './Header';
import ResourceTabs from './ResourceTabs';

function processData(data) {
    if (!data || !data.node) {
        return {
            name: '',
        };
    }

    const result = { ...data.node };
    const [ipAddress] = result.internalIpAddresses;
    result.ipAddress = ipAddress;

    const joinedAt = new Date(result.joinedAt);
    result.joinedAtDate = format(joinedAt, 'MM/DD/YYYY');
    result.joinedAtTime = format(joinedAt, 'h:mm:ss:A');
    return result;
}

const NodePage = ({
    entityId,
    listEntityType1,
    entityId1,
    entityType2,
    entityListType2,
    entityId2,
    query,
    sidePanelMode,
}) => {
    const [isExporting, setIsExporting] = useState(false);
    const searchParam = useContext(searchContext);
    return (
        <Query query={NODE_QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (isGQLLoading(loading, data)) {
                    return <Loader />;
                }

                if (!data.node) {
                    return (
                        <PageNotFound
                            resourceType={entityTypes.NODE}
                            useCase={useCases.COMPLIANCE}
                        />
                    );
                }
                const node = processData(data);
                const {
                    name,
                    id,
                    containerRuntimeVersion,
                    clusterName,
                    osImage,
                    ipAddress,
                    joinedAtDate,
                    joinedAtTime,
                    kernelVersion,
                    labels,
                } = node;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                if (listEntityType1 && !sidePanelMode) {
                    const listQuery = {
                        groupBy:
                            listEntityType1 === entityTypes.CONTROL ? entityTypes.STANDARD : '',
                        node: name,
                        ...query[searchParam],
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
                                        <IconWidget
                                            title="Container Runtime"
                                            icon={ContainerRuntime}
                                            description={containerRuntimeVersion}
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
                                            headline={osImage}
                                            description={kernelVersion}
                                            loading={loading}
                                        />
                                    </div>
                                    <div className="md:pl-3 pb-3">
                                        <InfoWidget
                                            title="Node Join Time"
                                            headline={joinedAtDate}
                                            description={joinedAtTime}
                                            loading={loading}
                                        />
                                    </div>
                                    <div className="md:pr-3 pt-3">
                                        <IconWidget
                                            title="IP Address"
                                            icon={IpAddress}
                                            description={ipAddress}
                                            loading={loading}
                                        />
                                    </div>
                                    <div className="md:pl-3 pt-3">
                                        <IconWidget
                                            title="Hostname"
                                            icon={Hostname}
                                            textSizeClass="text-base"
                                            description={name}
                                            loading={loading}
                                        />
                                    </div>
                                </div>

                                <Widget
                                    className={`sx-2 ${pdfClassName}`}
                                    header={`${labels.length} ${pluralize('Label', labels.length)}`}
                                >
                                    <Labels labels={labels} />
                                </Widget>
                                <ComplianceByStandards
                                    entityId={id}
                                    entityName={name}
                                    entityType={entityTypes.NODE}
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
                                    entityType={entityTypes.NODE}
                                    listEntityType={listEntityType1}
                                    entityName={name}
                                    entityId={id}
                                    isExporting={isExporting}
                                    setIsExporting={setIsExporting}
                                />
                                <ResourceTabs
                                    entityId={id}
                                    entityType={entityTypes.NODE}
                                    selectedType={listEntityType1}
                                    resourceTabs={[
                                        entityTypes.CONTROL,
                                        entityTypes.CLUSTER,
                                        entityTypes.NAMESPACE,
                                    ]}
                                />
                            </>
                        )}
                        {contents}
                        {isExporting && <BackdropExporting />}
                    </section>
                );
            }}
        </Query>
    );
};
NodePage.propTypes = entityPagePropTypes;
NodePage.defaultProps = entityPageDefaultProps;

export default NodePage;
