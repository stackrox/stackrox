import React from 'react';
import PropTypes from 'prop-types';
import { resourceTypes } from 'constants/entityTypes';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import { resourceLabels } from 'messages/common';
import { NODES_BY_CLUSTER } from 'queries/node';
import capitalize from 'lodash/capitalize';

import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import CountWidget from 'Components/CountWidget';

const queryMap = {
    [resourceTypes.NODE]: NODES_BY_CLUSTER
    // TODO: [resourceTypes.NAMESPACE] : NETWORK_POLICIES_BY_NAMESPACE
};

const ResourceCount = ({ entityType, params }) => {
    function getUrl(name) {
        const linkParams = {
            entityType
        };
        if (params.entityId && params.entityType) {
            linkParams.query = {
                [capitalize(params.entityType)]: name
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
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                const contents = <Loader />;
                const headerText = `${resourceLabels[entityType]} Count`;
                if (!loading && data && data.results) {
                    const url = getUrl(data.results.name);
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
    params: PropTypes.shape({}).isRequired
};

export default ResourceCount;
