import React from 'react';
import PropTypes from 'prop-types';
import LinkListWidget from 'Components/LinkListWidget';
import pageTypes from 'constants/pageTypes';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import AppLink from 'Components/AppLink';
import entityTypes from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import { resourceLabels } from 'messages/common';
import { RELATED_SECRETS, RELATED_DEPLOYMENTS, ALL_NAMESPACES } from 'queries/namespace';
import queryService from 'modules/queryService';
import { NODES_BY_CLUSTER } from 'queries/node';

const queryMap = {
    [entityTypes.NAMESPACE]: ALL_NAMESPACES,
    [entityTypes.NODE]: NODES_BY_CLUSTER,
    [entityTypes.SECRET]: RELATED_SECRETS,
    [entityTypes.DEPLOYMENT]: RELATED_DEPLOYMENTS
};

function getPageContext(entityType) {
    switch (entityType) {
        case entityTypes.DEPLOYMENT:
            return contextTypes.RISK;
        case entityTypes.SECRET:
            return contextTypes.SECRET;
        default:
            return contextTypes.COMPLIANCE;
    }
}

const ResourceRelatedEntitiesList = ({
    listEntityType,
    pageEntityType,
    pageEntity,
    clusterName,
    className,
    limit
}) => {
    const linkContext = getPageContext(listEntityType);
    const resourceLabel = resourceLabels[listEntityType];

    function processData(data) {
        if (!data || !data.results) return [];

        let items = data.results;
        if (listEntityType === entityTypes.NAMESPACE) {
            items = items
                .map(item => item.metadata)
                .filter(item => item.clusterName === pageEntity.name);
        }
        if (listEntityType === entityTypes.NODE) {
            items = data.results.nodes;
        }

        return items.map(item => ({
            label: item.name,
            link: URLService.getLinkTo(linkContext, pageTypes.ENTITY, {
                query: {},
                entityId: item.id,
                entityType: listEntityType
            }).url
        }));
    }

    const viewAllLink =
        pageEntity && pageEntity.id ? (
            <AppLink
                context={linkContext}
                pageType={pageTypes.LIST}
                params={{
                    query: {
                        [pageEntityType]: pageEntity.name,
                        [entityTypes.CLUSTER]: clusterName
                    },
                    entityType: listEntityType
                }}
                className="no-underline"
            >
                <button className="btn-sm btn-base btn-sm" type="button">
                    View All
                </button>
            </AppLink>
        ) : null;

    function getVariables() {
        if (listEntityType === entityTypes.NAMESPACE) {
            return null;
        }

        const variables = {
            query: queryService.objectToWhereClause({
                [pageEntityType]: pageEntity.name,
                [entityTypes.CLUSTER]: clusterName
            })
        };

        if (listEntityType === entityTypes.NODE) {
            variables.id = pageEntity.id;
        }

        return variables;
    }

    function getHeadline(list) {
        if (!list) return `Related ${pluralize(resourceLabel)}`;
        return `${list.length} Related ${pluralize(resourceLabel)}`;
    }

    return (
        <LinkListWidget
            query={queryMap[listEntityType]}
            variables={getVariables()}
            processData={processData}
            getHeadline={getHeadline}
            headerComponents={viewAllLink}
            className={className}
            limit={limit}
        />
    );
};

ResourceRelatedEntitiesList.propTypes = {
    listEntityType: PropTypes.string.isRequired,
    pageEntityType: PropTypes.string.isRequired,
    className: PropTypes.string,
    pageEntity: PropTypes.shape({
        id: PropTypes.string,
        name: PropTypes.string
    }),
    clusterName: PropTypes.string.isRequired,
    limit: PropTypes.number
};

ResourceRelatedEntitiesList.defaultProps = {
    pageEntity: null,
    className: null,
    limit: 20
};

export default ResourceRelatedEntitiesList;
