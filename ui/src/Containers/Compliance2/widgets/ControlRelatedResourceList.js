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
import { AGGREGATED_RESULTS as QUERY } from 'queries/controls';
import queryService from 'modules/queryService';

const ControlRelatedEntitiesList = ({
    listEntityType,
    pageEntityType,
    pageEntity,
    limit,
    standard
}) => {
    const linkContext = contextTypes.COMPLIANCE;

    function processData(data) {
        if (!data || !data.results) return [];

        const { clusters } = data;
        let options = clusters;
        if (listEntityType === entityTypes.NAMESPACE) {
            options = clusters
                .reduce((acc, cluster) => acc.concat(cluster.namespaces), [])
                .map(ns => ns.metadata);
        } else if (listEntityType === entityTypes.NODE) {
            options = clusters.reduce((acc, cluster) => acc.concat(cluster.nodes), []);
        }

        function getEntityName(id) {
            const match = options.find(item => item.id === id);
            return match ? match.name : null;
        }

        const ids = data.results.results
            .filter(item => item.numPassing > 0 || item.numFailing > 0)
            .map(item => item.aggregationKeys.find(key => key.scope === listEntityType).id);

        const filteredIds = [];
        ids.forEach(id => {
            if (!filteredIds.includes(id)) filteredIds.push(id);
        });

        return filteredIds.map(id => ({
            label: getEntityName(id),
            link: URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.ENTITY, {
                entityId: id,
                query: { standard, [pageEntityType]: pageEntity.name },
                entityType: listEntityType
            }).url
        }));
    }

    function getHeadline(items) {
        if (!items) return 'Loading...';
        const resourceLabel = resourceLabels[listEntityType];
        const count = typeof items.length !== 'undefined' ? items.length : 0;
        return `${count} Related ${pluralize(resourceLabel, count)}`;
    }

    const viewAllLink =
        pageEntity && pageEntity.id && listEntityType !== entityTypes.SECRET ? (
            <AppLink
                context={linkContext}
                pageType={pageTypes.LIST}
                params={{
                    query: { standard, [pageEntityType]: pageEntity.name },
                    entityType: listEntityType
                }}
                className="no-underline"
            >
                <button className="btn-sm btn-base btn-sm" type="button">
                    View All
                </button>
            </AppLink>
        ) : null;

    const variables = {
        groupBy: [pageEntityType, listEntityType],
        unit: entityTypes.CONTROL,
        where: queryService.objectToWhereClause({ [pageEntityType]: pageEntity.name })
    };

    return (
        <LinkListWidget
            query={QUERY}
            className="sx-2"
            variables={variables}
            processData={processData}
            getHeadline={getHeadline}
            headerComponents={viewAllLink}
            limit={limit}
        />
    );
};

ControlRelatedEntitiesList.propTypes = {
    listEntityType: PropTypes.string.isRequired,
    pageEntityType: PropTypes.string.isRequired,
    pageEntity: PropTypes.shape({
        id: PropTypes.string,
        name: PropTypes.string
    }),
    limit: PropTypes.number,
    standard: PropTypes.string.isRequired
};

ControlRelatedEntitiesList.defaultProps = {
    pageEntity: null,
    limit: 10
};

export default ControlRelatedEntitiesList;
