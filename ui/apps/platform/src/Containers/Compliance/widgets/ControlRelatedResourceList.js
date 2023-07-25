import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import URLService from 'utils/URLService';
import entityTypes from 'constants/entityTypes';
import useCases from 'constants/useCaseTypes';
import { AGGREGATED_RESULTS as QUERY } from 'queries/controls';
import queryService from 'utils/queryService';
import ReactRouterPropTypes from 'react-router-prop-types';
import { withRouter, Link } from 'react-router-dom';
import searchContext from 'Containers/searchContext';

import { entityNounOrdinaryCase } from '../entitiesForCompliance';
import LinkListWidget from './LinkListWidget';

const ControlRelatedEntitiesList = ({
    match,
    location,
    listEntityType,
    pageEntityType,
    pageEntity,
    limit,
    standard,
    className,
}) => {
    const linkContext = useCases.COMPLIANCE;
    const searchParam = useContext(searchContext);

    function processData(data) {
        if (!data || !data.results) {
            return [];
        }

        const { clusters } = data;
        let options = clusters;
        if (listEntityType === entityTypes.NAMESPACE) {
            options =
                clusters?.reduce(
                    (acc, cluster) =>
                        acc.concat(
                            cluster.namespaces.map((ns) => ({
                                ...ns.metadata,
                                name: `${cluster?.name}/${ns?.metadata?.name}`,
                            }))
                        ),
                    []
                ) || [];
        } else if (listEntityType === entityTypes.NODE) {
            options =
                clusters?.reduce(
                    (acc, cluster) =>
                        acc.concat(
                            cluster.nodes.map((node) => ({
                                ...node,
                                name: `${cluster?.name}/${node?.name}`,
                            }))
                        ),
                    []
                ) || [];
        } else if (listEntityType === entityTypes.DEPLOYMENT) {
            options = data.deployments || [];
        }

        function getEntityName(id) {
            const found = options.find((item) => item.id === id);
            return found ? found.name : null;
        }

        const ids = data.results.results
            .filter((item) => item.numPassing > 0 || item.numFailing > 0)
            .map((item) => item.aggregationKeys.find((key) => key.scope === listEntityType).id);

        const filteredIds = [];
        ids.forEach((id) => {
            if (!filteredIds.includes(id)) {
                filteredIds.push(id);
            }
        });

        const result = filteredIds.map((id) => ({
            label: getEntityName(id),
            link: URLService.getURL(match, location).base(listEntityType, id).url(),
        }));

        return result;
    }

    function getHeadline(items) {
        if (!items) {
            return 'Loading...';
        }
        const count = typeof items.length !== 'undefined' ? items.length : 0;
        return `${count} related ${entityNounOrdinaryCase(count, listEntityType)}`;
    }

    const viewAllLink =
        pageEntity && pageEntity.id && listEntityType !== entityTypes.SECRET ? (
            <Link
                to={URLService.getURL(match, location)
                    .base(listEntityType, null, linkContext)
                    .query({ [searchParam]: { standard, [pageEntityType]: pageEntity?.name } })
                    .url()}
                className="no-underline"
            >
                <button className="btn-sm btn-base btn-sm" type="button">
                    View All
                </button>
            </Link>
        ) : null;

    const variables = {
        groupBy: [pageEntityType, listEntityType],
        unit: entityTypes.CONTROL,
        where: queryService.objectToWhereClause({ [`${pageEntityType} ID`]: pageEntity.id }),
    };

    return (
        <LinkListWidget
            query={QUERY}
            className={`sx-2 ${className}`}
            variables={variables}
            processData={processData}
            getHeadline={getHeadline}
            headerComponents={viewAllLink}
            limit={limit}
            id="related-resource-list"
        />
    );
};

ControlRelatedEntitiesList.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    listEntityType: PropTypes.string.isRequired,
    pageEntityType: PropTypes.string.isRequired,
    pageEntity: PropTypes.shape({
        id: PropTypes.string,
        name: PropTypes.string,
    }),
    limit: PropTypes.number,
    standard: PropTypes.string.isRequired,
    className: PropTypes.string,
};

ControlRelatedEntitiesList.defaultProps = {
    pageEntity: null,
    limit: 10,
    className: '',
};

export default withRouter(ControlRelatedEntitiesList);
