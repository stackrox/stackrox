import React from 'react';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import { resourceTypes } from 'constants/entityTypes';
import CountWidget from 'Components/CountWidget';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import { resourceLabels } from 'messages/common';
import { NODES_BY_CLUSTER } from '../../../queries/node';

const queryMap = {
    [resourceTypes.NODE]: NODES_BY_CLUSTER
    // TODO: [resourceTypes.NAMESPACE] : NETWORK_POLICIES_BY_NAMESPACE
};

const ResourceCount = ({ entityType, params, loading: parentLoading }) => {
    function getUrl() {
        const linkParams = {
            entityType
        };
        if (params.entityId && params.entityType) {
            linkParams.query = {
                [params.entityType]: params.entityId
            };
        }
        return URLService.getLinkTo(params.context, pageTypes.LIST, linkParams).url;
    }

    function processData(data) {
        if (entityType === resourceTypes.NODE) {
            return data.nodes.length;
        }
        if (entityType === resourceTypes.NAMESPACE) {
            return data.namespaces.length;
        }
        return 0;
    }

    const query = queryMap[entityType];
    const variables = { id: params.entityId };

    return (
        <Query query={query} variables={variables} pollInterval={5000}>
            {({ loading, data }) => {
                const contents = <Loader />;
                const headerText = `${resourceLabels[entityType]} Count`;
                if (!loading && !parentLoading && data && data.results) {
                    const url = getUrl(entityType, params);
                    const count = processData(data.results);
                    return <CountWidget title={headerText} count={count} linkUrl={url} />;
                }
                return (
                    <Widget header={headerText} bodyClassName="p-2">
                        {contents}
                    </Widget>
                );
            }}
        </Query>
    );
};

ResourceCount.propTypes = {
    entityType: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired,
    loading: PropTypes.bool
};

ResourceCount.defaultProps = {
    loading: false
};

export default ResourceCount;
