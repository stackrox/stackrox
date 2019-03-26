import React from 'react';
import PropTypes from 'prop-types';
import entityTypes, { searchCategories } from 'constants/entityTypes';
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
import uniq from 'lodash/uniq';

const ResourceCount = ({ entityType, relatedToResourceType, relatedToResource }) => {
    function getUrl() {
        const linkParams = {
            entityType: relatedToResourceType,
            entityId: relatedToResource.id,
            listEntityType: entityType
        };

        // TODO: Remove this when deployments and secrets are brought into main framework.
        let context = contextTypes.COMPLIANCE;
        if (entityType === entityTypes.SECRET) context = contextTypes.SECRET;
        else if (entityType === entityTypes.DEPLOYMENT) context = contextTypes.RISK;
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

    function getCount(data) {
        const searchCategory = searchCategories[entityType];

        if (entityType === entityTypes.CONTROL) {
            return uniq(
                data.aggregatedResults.results
                    .filter(datum => datum.numFailing + datum.numPassing)
                    .map(datum => datum.aggregationKeys[0].id)
            ).length;
        }

        return uniq(
            data.search.filter(datum => datum.category === searchCategory).map(datum => datum.id)
        ).length;
    }

    const variables = getVariables();

    return (
        <Query query={QUERY} variables={variables}>
            {({ loading, data }) => {
                const contents = <Loader />;
                const headerText = `${resourceLabels[entityType]} Count`;
                if (!loading && data && data.search) {
                    const count = getCount(data);
                    const url = getUrl();
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
    entityType: PropTypes.string,
    relatedToResourceType: PropTypes.string.isRequired,
    relatedToResource: PropTypes.shape({})
};

ResourceCount.defaultProps = {
    entityType: null,
    relatedToResource: null
};

export default ResourceCount;
