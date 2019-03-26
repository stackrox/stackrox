import React from 'react';
import PropTypes from 'prop-types';
import { Link, withRouter } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import URLService from 'modules/URLService';
import contexts from 'constants/contextTypes';
import pageTypes from 'constants/pageTypes';
import { resourceLabels } from 'messages/common';
import pluralize from 'pluralize';
import Query from 'Components/ThrowingQuery';
import { SEARCH_WITH_CONTROLS as QUERY } from 'queries/search';
import uniq from 'lodash/uniq';
import entityTypes, { searchCategories } from 'constants/entityTypes';
import queryService from 'modules/queryService';

const ResourceTabs = ({ entityType, entityId, resourceTabs, match }) => {
    function getLinkToListType(listEntityType) {
        const urlParams = {
            entityId,
            entityType,
            ...(listEntityType ? { listEntityType } : {})
        };
        return URLService.getLinkTo(contexts.COMPLIANCE, pageTypes.ENTITY, urlParams).url;
    }

    function getCount(type, data) {
        const searchCategory = searchCategories[type];

        if (type === entityTypes.CONTROL) {
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

    function processData(data) {
        const tabData = [
            {
                title: 'overview',
                link: getLinkToListType()
            }
        ];

        if (resourceTabs.length && data && data.search) {
            resourceTabs.forEach(type => {
                const count = getCount(type, data);
                if (count > 0)
                    tabData.push({
                        title: `${count} ${pluralize(resourceLabels[type], count)}`,
                        link: getLinkToListType(type)
                    });
            });
        }
        return tabData;
    }

    function getVariables() {
        return {
            query: queryService.objectToWhereClause({ [`${entityType} ID`]: entityId }),
            categories: [searchCategories.NODE, searchCategories.NAMESPACE]
        };
    }

    const variables = getVariables();
    return (
        <Query query={QUERY} variables={variables}>
            {({ data }) => {
                const tabData = processData(data);
                return (
                    <ul className="border-t border-b border-primary-400 list-reset pl-3">
                        {tabData.map((datum, i) => {
                            const borderLeft = !i ? 'border-l' : '';
                            let bgColor = 'bg-primary-200';
                            let textColor = 'text-base-600';
                            if (datum.link === match.url) {
                                bgColor = 'bg-base-100';
                                textColor = 'text-primary-600';
                            }

                            return (
                                <li key={datum.title} className="inline-block">
                                    <Link
                                        className={`no-underline ${textColor} ${borderLeft} ${bgColor} border-r border-primary-400  w-32 text-center pt-3 pb-3 uppercase tracking-widest inline-block`}
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
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    resourceTabs: PropTypes.arrayOf(PropTypes.string),
    match: ReactRouterPropTypes.match.isRequired
};

ResourceTabs.defaultProps = {
    resourceTabs: []
};

export default withRouter(ResourceTabs);
