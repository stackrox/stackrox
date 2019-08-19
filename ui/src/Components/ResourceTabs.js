import React from 'react';
import PropTypes from 'prop-types';
import { Link, withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { resourceLabels } from 'messages/common';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import Query from 'Components/ThrowingQuery';
import { SEARCH_WITH_CONTROLS as QUERY } from 'queries/search';
import queryService from 'modules/queryService';
import {
    getResourceCountFromAggregatedResults,
    getResourceCountFromComplianceResults
} from 'modules/complianceUtils';
import entityTypes from 'constants/entityTypes';

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
                title: 'overview',
                link: getLinkToListType(),
                type: null
            }
        ];

        if (resourceTabs.length && data) {
            resourceTabs.forEach(type => {
                let count;
                if (entityType === entityTypes.CONTROL) {
                    count = getResourceCountFromComplianceResults(type, data);
                } else {
                    count = getResourceCountFromAggregatedResults(type, data);
                }
                if (count > 0)
                    tabData.push({
                        title: `${count} ${pluralize(resourceLabels[type], count)}`,
                        link: getLinkToListType(type),
                        type
                    });
            });
        }

        return tabData;
    }

    function getVariables() {
        // entitytype.NODE and namespace don't play well in groupBy
        return {
            query: queryService.objectToWhereClause({ [`${entityType} ID`]: entityId })
        };
    }

    return (
        <Query query={QUERY} variables={getVariables()}>
            {({ loading, data }) => {
                if (loading) return null;
                const tabData = processData(data);
                return (
                    <ul className="border-b border-base-400 bg-base-200 list-reset pl-3">
                        {tabData.map((datum, i) => {
                            const borderLeft = !i ? 'border-l' : '';
                            let bgColor = 'bg-base-200';
                            let textColor = 'text-base-600';
                            const style = {
                                borderColor: 'hsla(225, 44%, 87%, 1)'
                            };
                            if (datum.type === selectedType || (!selectedType && !datum.type)) {
                                bgColor = 'bg-base-100';
                                textColor = 'text-primary-600';
                                style.borderTopColor = 'hsla(225, 90%, 67%, 1)';
                                style.borderTopWidth = '1px';
                            }

                            return (
                                // eslint-disable-next-line
                                <li
                                    key={datum.title}
                                    className="inline-block"
                                >
                                    <Link
                                        style={style}
                                        className={`no-underline ${textColor} ${borderLeft} ${bgColor} border-r min-w-32 px-3 text-center pt-3 pb-3 uppercase tracking-widest inline-block`}
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
    resourceTabs: PropTypes.arrayOf(PropTypes.string)
};

ResourceTabs.defaultProps = {
    resourceTabs: PropTypes.arrayOf(PropTypes.string),
    selectedType: null
};

export default withRouter(ResourceTabs);
