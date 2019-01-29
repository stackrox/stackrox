import React from 'react';
import componentTypes from 'constants/componentTypes';
import Widget from 'Components/Widget';
import Query from 'Components/AppQuery';
import Loader from 'Components/Loader';
import { resourceTypes } from 'constants/entityTypes';
import labels from 'messages/common';
import PropTypes from 'prop-types';
import AppLink from 'Components/AppLink';

const pageToWidgetEntityMap = {
    [resourceTypes.CLUSTERS]: resourceTypes.DEPLOYMENTS,
    [resourceTypes.NAMESPACES]: resourceTypes.DEPLOYMENTS,
    [resourceTypes.NODES]: resourceTypes.DEPLOYMENTS
};

const entityTypeToNameMap = {
    [resourceTypes.NAMESPACES]: `${labels.resourceLabels.NAMESPACE}S`,
    [resourceTypes.DEPLOYMENTS]: `${labels.resourceLabels.DEPLOYMENTS}S`
};

const RelatedEntitiesList = ({ params }) => {
    const { entityType: pageEntityType, pageType, context } = params;
    const widgetEntityType = pageToWidgetEntityMap[pageEntityType];
    const entityTypeText = entityTypeToNameMap[widgetEntityType];

    function processData(results) {
        if (!results) return [];

        const dataWithLink = results[widgetEntityType].map(item => {
            const linkParams = {
                query: params.query,
                entityId: item.id,
                entityType: widgetEntityType
            };
            const link = (
                <AppLink
                    context={context}
                    pageType={pageType}
                    entityType={widgetEntityType}
                    params={linkParams}
                >
                    {item.name}
                </AppLink>
            );
            return {
                ...item,
                link
            };
        });

        return dataWithLink;
    }
    return (
        <Query params={params} componentType={componentTypes.RELATED_ENTITIES_LIST}>
            {({ loading, data }) => {
                let contents = <Loader />;
                let headerText = `Related ${entityTypeText}`;
                if (!loading && data && data.results) {
                    const results = processData(data.results);
                    headerText = `${results.length} Related ${entityTypeText}`;

                    contents = (
                        <ul>
                            {results.map(entity => (
                                <li key={entity.id}>{entity.link}</li>
                            ))}
                        </ul>
                    );
                }

                return <Widget header={headerText}>{contents}</Widget>;
            }}
        </Query>
    );
};

RelatedEntitiesList.propTypes = {
    params: PropTypes.shape({
        entityType: PropTypes.string,
        context: PropTypes.string,
        pageType: PropTypes.string
    }).isRequired
};

export default RelatedEntitiesList;
