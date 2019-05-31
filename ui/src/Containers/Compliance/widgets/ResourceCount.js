import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';
import URLService from 'modules/URLService';
import pageTypes from 'constants/pageTypes';
import { resourceLabels } from 'messages/common';
import capitalize from 'lodash/capitalize';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import CountWidget from 'Components/CountWidget';
import contextTypes from 'constants/contextTypes';
import { SEARCH_WITH_CONTROLS as QUERY } from 'queries/search';
import queryService from 'modules/queryService';
import { getResourceCountFromResults } from 'modules/complianceUtils';

const ResourceCount = ({ entityType, relatedToResourceType, relatedToResource, count }) => {
    function getUrl() {
        const linkParams = {
            entityType: relatedToResourceType,
            entityId: relatedToResource.id,
            listEntityType: entityType
        };

        // TODO: Remove this when deployments and secrets are brought into main framework.
        let context = contextTypes.COMPLIANCE;
        if (entityType === entityTypes.SECRET) context = contextTypes.SECRET;
        if (context !== contextTypes.COMPLIANCE) {
            return URLService.getLinkTo(context, pageTypes.LIST, {
                ...linkParams,
                query: { [`${capitalize(relatedToResourceType)}`]: relatedToResource.name }
            }).url;
        }

        return URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.ENTITY, linkParams).url;
    }

    function getVariables() {
        let query;
        switch (relatedToResourceType) {
            case entityTypes.NAMESPACE:
                query = {
                    namespace: relatedToResource.name,
                    cluster: relatedToResource.clusterName
                };
                break;
            default:
                query = { [`${capitalize(relatedToResourceType)} ID`]: relatedToResource.id };
        }

        return {
            query: queryService.objectToWhereClause(query),
            categories: []
        };
    }

    const variables = getVariables();
    const headerText = `${resourceLabels[entityType]} Count`;
    const url = getUrl();

    if (count || count === 0) {
        return (
            <Widget header={headerText} bodyClassName="p-2">
                <CountWidget title={headerText} count={count} linkUrl={url} />;
            </Widget>
        );
    }

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                const contents = <Loader />;

                if (!loading && data) {
                    const resourceCount = getResourceCountFromResults(entityType, data);
                    return <CountWidget title={headerText} count={resourceCount} linkUrl={url} />;
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
    entityType: PropTypes.string,
    relatedToResourceType: PropTypes.string.isRequired,
    relatedToResource: PropTypes.shape({}),
    count: PropTypes.number
};

ResourceCount.defaultProps = {
    entityType: null,
    relatedToResource: null,
    count: null
};

export default ResourceCount;
