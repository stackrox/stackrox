import React from 'react';
import PropTypes from 'prop-types';
import { Link, withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { resourceLabels } from 'messages/common';
import URLService from 'utils/URLService';
import pluralize from 'pluralize';
import Query from 'Components/ThrowingQuery';
import { SEARCH_WITH_CONTROLS as QUERY } from 'queries/search';
import queryService from 'utils/queryService';
import { getResourceCountFromAggregatedResults } from 'utils/complianceUtils';

const ResourceTabs = ({ entityType, entityId, resourceTabs, selectedType, match, location }) => {
    function getLinkToListType(listEntityType) {
        return URLService.getURL(match, location)
            .base(entityType, entityId)
            .push(listEntityType)
            .query()
            .url();
    }

    function processData(data) {
        const tabData = [
            {
                title: 'Overview',
                link: getLinkToListType(),
                type: null,
            },
        ];

        if (resourceTabs.length && data) {
            resourceTabs.forEach((type) => {
                const count = getResourceCountFromAggregatedResults(type, data);
                if (count > 0) {
                    tabData.push({
                        title: `${count} ${pluralize(resourceLabels[type], count)}`,
                        link: getLinkToListType(type),
                        type,
                    });
                }
            });
        }

        return tabData;
    }

    function getVariables() {
        // entitytype.NODE and namespace don't play well in groupBy
        return {
            query: queryService.objectToWhereClause({ [`${entityType} ID`]: entityId }),
        };
    }

    return (
        <Query query={QUERY} variables={getVariables()}>
            {({ loading, data }) => {
                if (loading) {
                    return null;
                }
                const tabData = processData(data);
                return (
                    <ul className="border-b border-base-400 bg-base-100 pl-3">
                        {tabData.map((datum, i) => {
                            const borderLeft = !i ? 'border-l' : '';
                            const bgColor =
                                datum.type === selectedType || (!selectedType && !datum.type)
                                    ? 'bg-primary-200'
                                    : 'bg-base-100';
                            const textColor = 'text-base-600';

                            return (
                                <li key={datum.title} className="inline-block">
                                    <Link
                                        className={`no-underline ${textColor} ${borderLeft} ${bgColor} border-r border-base-400 min-w-32 px-3 text-center pt-3 pb-3 font-700 inline-block`}
                                        to={datum.link}
                                    >
                                        {datum.title}
                                    </Link>
                                </li>
                            );
                        })}
                    </ul>
                );
            }}
        </Query>
    );
};

ResourceTabs.propTypes = {
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired,
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    selectedType: PropTypes.string,
    resourceTabs: PropTypes.arrayOf(PropTypes.string),
};

ResourceTabs.defaultProps = {
    resourceTabs: PropTypes.arrayOf(PropTypes.string),
    selectedType: null,
};

export default withRouter(ResourceTabs);
