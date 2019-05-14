import React from 'react';
import PropTypes from 'prop-types';
import entityTypes, { searchCategories as searchCategoryTypes } from 'constants/entityTypes';
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
import Loader from 'Components/Loader';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter } from 'react-router-dom';
import URLService from 'modules/URLService';
import ResourceTabs from 'Components/ResourceTabs';
import ComplianceList from 'Containers/Compliance/List/List';
import PageNotFound from 'Components/PageNotFound';
import ControlAssessment from 'Containers/Compliance/widgets/ControlAssessment';
import ComplianceWidgetsGroup from 'Containers/Compliance/ComplianceWidgetsGroup';
import Header from './Header';
import SearchInput from '../SearchInput';

function processData(data) {
    if (!data || !data.node)
        return {
            name: ''
        };

    const result = { ...data.node };
    const [ipAddress] = result.internalIpAddresses;
    result.ipAddress = ipAddress;

    const joinedAt = new Date(result.joinedAt);
    result.joinedAtDate = format(joinedAt, 'MM/DD/YYYY');
    result.joinedAtTime = format(joinedAt, 'h:mm:ss:A');
    return result;
}

const NodePage = ({ match, location, nodeId, sidePanelMode, controlResult }) => {
    const params = URLService.getParams(match, location);
    const entityId = nodeId || params.entityId;
    const listEntityType = URLService.getEntityTypeKeyFromValue(params.listEntityType);

    return (
        <Query query={NODE_QUERY} variables={{ id: entityId }}>
            {({ loading, data }) => {
                if (loading) return <Loader />;
                if (!data.node) return <PageNotFound resourceType={entityTypes.NODE} />;
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
                    labels
                } = node;
                const pdfClassName = !sidePanelMode ? 'pdf-page' : '';
                let contents;

                const searchComponent = listEntityType ? (
                    <SearchInput categories={[searchCategoryTypes[listEntityType]]} />
                ) : null;

                if (listEntityType && !sidePanelMode) {
                    const queryParams = { ...params.query };
                    queryParams.node = name;
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

                                <ControlAssessment
                                    className={`sx-2 ${pdfClassName}`}
                                    controlResult={controlResult}
                                />

                                <ComplianceWidgetsGroup
                                    controlResult={controlResult}
                                    entityType={entityTypes.NODE}
                                    entityName={name}
                                    entityId={id}
                                    pdfClassName={pdfClassName}
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
                                    listEntityType={listEntityType}
                                    entityName={name}
                                    entityId={id}
                                />
                                <ResourceTabs
                                    entityId={id}
                                    entityType={entityTypes.NODE}
                                    resourceTabs={[entityTypes.CONTROL, entityTypes.NAMESPACE]}
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
NodePage.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    nodeId: PropTypes.string,
    sidePanelMode: PropTypes.bool,
    controlResult: PropTypes.shape({})
};

NodePage.defaultProps = {
    nodeId: null,
    sidePanelMode: false,
    controlResult: null
};

export default withRouter(NodePage);
