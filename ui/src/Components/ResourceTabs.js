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
import queryService from 'modules/queryService';
import { getResourceCountFromResults } from 'modules/complianceUtils';

const ResourceTabs = ({ entityType, entityId, standardId, resourceTabs, match }) => {
    function getLinkToListType(listEntityType) {
        const urlParams = {
            entityId,
            entityType,
            standardId,
            controlId: entityId,
            ...(listEntityType ? { listEntityType } : {})
        };
        return URLService.getLinkTo(contexts.COMPLIANCE, pageTypes.ENTITY, urlParams).url;
    }

    function processData(data) {
        const tabData = [
            {
                title: 'overview',
                link: getLinkToListType()
            }
        ];

        if (resourceTabs.length && data) {
            resourceTabs.forEach(type => {
                const count = getResourceCountFromResults(type, data);
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
        // entitytype.NODE and namespace don't play well in groupBy
        return {
            query: queryService.objectToWhereClause({ [`${entityType} ID`]: entityId })
        };
    }

    const variables = getVariables();
    return (
        <Query query={QUERY} variables={variables}>
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
                            if (datum.link === match.url) {
                                bgColor = 'bg-base-100';
                                textColor = 'text-primary-600';
                                style.borderTopColor = 'hsla(225, 90%, 67%, 1)';
                                style.borderTopWidth = '1px';
                            }

                            return (
                                <li key={datum.title} className="inline-block">
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
    entityType: PropTypes.string.isRequired,
    entityId: PropTypes.string.isRequired,
    standardId: PropTypes.string,
    resourceTabs: PropTypes.arrayOf(PropTypes.string),
    match: ReactRouterPropTypes.match.isRequired
};

ResourceTabs.defaultProps = {
    resourceTabs: [],
    standardId: null
};

export default withRouter(ResourceTabs);
