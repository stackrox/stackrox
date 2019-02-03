import React from 'react';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import { resourceTypes } from 'constants/entityTypes';
import CountWidget from 'Components/CountWidget';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import pluralize from 'pluralize';
import { NODES_BY_CLUSTER } from '../../../queries/node';

const queryMap = {
    [resourceTypes.NODES]: NODES_BY_CLUSTER
    // TODO: [resourceTypes.NAMESPACES] : NETWORK_POLICIES_BY_NAMESPACE
};

const ResourceCount = ({ type, params }) => {
    function getUrl() {
        const linkParams = {
            entityType: type
        };
        if (params.entityId && params.entityType) {
            linkParams.query = {
                [pluralize.singular(params.entityType)]: params.entityId
            };
        }
        return URLService.getLinkTo(params.context, pageTypes.LIST, linkParams).url;
    }

    function processData(data) {
        if (type === resourceTypes.NODES) {
            return data.nodes.length;
        }
        if (type === resourceTypes.NAMESPACES) {
            return data.namespaces.length;
        }
        return 0;
    }

    const query = queryMap[type];
    const variables = { id: params.entityId };

    return (
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                const contents = <Loader />;
                const headerText = `${pluralize.singular(type)} Count`;
                if (!loading && data && data.results) {
                    const url = getUrl(type, params);
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
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({}).isRequired
};

export default ResourceCount;
