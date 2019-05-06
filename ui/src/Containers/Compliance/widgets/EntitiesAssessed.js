import React from 'react';
import PropTypes from 'prop-types';
import Raven from 'raven-js';
import contexts from 'constants/contextTypes';
import entityTypes from 'constants/entityTypes';
import pluralize from 'pluralize';
import upperCase from 'lodash/upperCase';
import {
    CLUSTERS_LIST_QUERY,
    NAMESPACES_LIST_QUERY,
    NODES_LIST_QUERY,
    DEPLOYMENTS_LIST_QUERY
} from 'queries/table';
import URLService from 'modules/URLService';
import { Link } from 'react-router-dom';

import Widget from 'Components/Widget';
import Message from 'Components/Message';
import Query from 'Components/ThrowingQuery';
import Loader from 'Components/Loader';
import queryService from 'modules/queryService';
import pageTypes from 'constants/pageTypes';

const getQuery = resourceEntityType => {
    let query;
    switch (upperCase(resourceEntityType)) {
        case entityTypes.CLUSTER:
            query = CLUSTERS_LIST_QUERY;
            break;
        case entityTypes.NAMESPACE:
            query = NAMESPACES_LIST_QUERY;
            break;
        case entityTypes.NODE:
            query = NODES_LIST_QUERY;
            break;
        case entityTypes.DEPLOYMENT:
            query = DEPLOYMENTS_LIST_QUERY;
            break;
        default:
            break;
    }
    return query;
};

const getVariables = controlName => ({
    where: queryService.objectToWhereClause({ Control: controlName })
});

const EntitiesAssessed = ({ className, controlResult }) => {
    if (!controlResult) return null;
    const controlId = controlResult.control.id;
    const controlName = controlResult.control.name;
    // eslint-disable-next-line
    const resourceEntityType = controlResult.resource.__typename;
    const QUERY = getQuery(resourceEntityType);
    const variables = getVariables(controlName);
    return (
        <Widget
            header={`All ${pluralize(resourceEntityType)} Assessed By This Control`}
            bodyClassName="flex-col"
            className={className}
            id="entity-assessed"
        >
            <div className="flex h-full w-full items-center justify-center">
                <div className="flex w-2/3 h-full border-r border-base-300">
                    <Message
                        type="info"
                        message={`View all the ${pluralize(
                            resourceEntityType
                        )} that have been assessed by this control across your entire environment.`}
                    />
                </div>
                <Query query={QUERY} variables={variables}>
                    {({ loading, data }) => {
                        if (loading) return <Loader />;
                        let contents;
                        try {
                            const numResources = data.results.results.length;
                            const to = URLService.getLinkTo(contexts.COMPLIANCE, pageTypes.ENTITY, {
                                entityType: entityTypes.CONTROL,
                                controlId,
                                listEntityType: upperCase(resourceEntityType)
                            });
                            contents = (
                                <Link
                                    className="flex h-full justify-center items-center w-1/3 text-6xl text-primary-700 font-400 px-4 text-center hover:bg-base-200 text-primary-600 no-underline"
                                    to={to}
                                >
                                    {numResources}
                                </Link>
                            );
                        } catch (error) {
                            Raven.captureException(error);
                            contents = (
                                <div className="flex h-full justify-center items-center w-1/3 text-6xl text-primary-700 font-400 px-4 text-center hover:bg-base-200">
                                    N/A
                                </div>
                            );
                        }
                        return contents;
                    }}
                </Query>
            </div>
        </Widget>
    );
};

EntitiesAssessed.propTypes = {
    className: PropTypes.string,
    controlResult: PropTypes.shape({})
};

EntitiesAssessed.defaultProps = {
    className: '',
    controlResult: null
};

export default EntitiesAssessed;
