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
import contextTypes from 'constants/contextTypes';

const queryMap = {
    [resourceTypes.NODE]: NODES_BY_CLUSTER
    // TODO: [resourceTypes.NAMESPACE] : NETWORK_POLICIES_BY_NAMESPACE
};

const ResourceCount = ({ resourceType, relatedToResourceType, relatedToResourceId }) => {
    function getUrl(name) {
        const linkParams = {
            entityType: resourceType
        };
        if (relatedToResourceId && relatedToResourceType) {
            linkParams.query = {
                [capitalize(relatedToResourceType)]: name
            };
        }
        return URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, linkParams).url;
    }

    function processData(data) {
        if (resourceType === resourceTypes.NODE) {
            return data.nodes.length;
        }
        if (resourceType === resourceTypes.NAMESPACE) {
            return data.namespaces.length;
        }
        return 0;
    }

    const query = queryMap[resourceType];
    const variables = { id: relatedToResourceId };

    return (
        <Query query={query} variables={variables}>
            {({ loading, data }) => {
                const contents = <Loader />;
                const headerText = `${resourceLabels[resourceType]} Count`;
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
    resourceType: PropTypes.string,
    relatedToResourceType: PropTypes.string.isRequired,
    relatedToResourceId: PropTypes.string
};

ResourceCount.defaultProps = {
    resourceType: null,
    relatedToResourceId: null
};

export default ResourceCount;
