import React from 'react';
import componentTypes from 'constants/componentTypes';
import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import PropTypes from 'prop-types';
import LinkListWidget from 'Components/LinkListWidget';
import pageTypes from 'constants/pageTypes';
import URLService from 'modules/URLService';
import pluralize from 'pluralize';
import { Link } from 'react-router-dom';
import entityTypes from 'constants/entityTypes';
import contextTypes from 'constants/contextTypes';
import NoResultsMessage from 'Components/NoResultsMessage';
import { resourceLabels } from 'messages/common';

const RelatedEntitiesList = ({ type, params }) => {
    const { entityType: pageEntityType, entityId: pageEntityId } = params;
    const linkContext =
        type === entityTypes.DEPLOYMENT ? contextTypes.RISK : contextTypes.COMPLIANCE;
    function processData(results) {
        if (!results) return [];

        const resultsProp = pluralize.plural(resourceLabels[type]);

        const dataWithLink = results[resultsProp].map(item => {
            const linkParams = {
                query: params.query,
                entityId: item.id,
                deploymentId: item.id,
                entityType: type
            };

            return {
                name: item.name,
                link: URLService.getLinkTo(linkContext, pageTypes.ENTITY, linkParams).url
            };
        });

        return dataWithLink;
    }

    let viewAllUrl = '/';
    const query = { [pageEntityType]: pageEntityId };
    if (type === entityTypes.DEPLOYMENT) {
        viewAllUrl = URLService.getLinkTo(contextTypes.RISK, pageTypes.LIST, {
            query
        });
    } else {
        viewAllUrl = URLService.getLinkTo(contextTypes.COMPLIANCE, pageTypes.LIST, {
            entityType: type,
            query
        }).url;
    }

    const viewAllLink = (
        <Link to={viewAllUrl} className="no-underline">
            <button className="btn-sm btn-base btn-sm" type="button">
                View All
            </button>
        </Link>
    );

    return (
        <Query params={params} componentType={componentTypes.RELATED_ENTITIES_LIST}>
            {({ loading, data }) => {
                let headerText = `Related ${pluralize(type)}`;
                let widget = (
                    <Widget header={headerText}>
                        <Loader />
                    </Widget>
                );
                if (!loading && data) {
                    if (data.results) {
                        const results = processData(data.results);
                        const entityTypeText = pluralize(type, results.length);
                        headerText = `${results.length} Related ${entityTypeText}`;
                        widget = (
                            <LinkListWidget
                                title={headerText}
                                data={results}
                                limit={7}
                                headerComponents={viewAllLink}
                            />
                        );
                    } else {
                        widget = (
                            <Widget header={headerText}>
                                <NoResultsMessage message="No data available" />
                            </Widget>
                        );
                    }
                }

                return widget;
            }}
        </Query>
    );
};

RelatedEntitiesList.propTypes = {
    type: PropTypes.string.isRequired,
    params: PropTypes.shape({
        entityType: PropTypes.string,
        context: PropTypes.string,
        pageType: PropTypes.string
    }).isRequired
};

export default RelatedEntitiesList;
