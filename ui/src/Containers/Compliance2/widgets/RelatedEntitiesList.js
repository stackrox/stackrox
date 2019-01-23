import React from 'react';
import componentTypes from 'constants/componentTypes';
import Widget from 'Components/Widget';
import Query from 'Components/ThrowingQuery';
import { withRouter } from 'react-router-dom';
import Loader from 'Components/Loader';
import ReactRouterPropTypes from 'react-router-prop-types';
import queryService from 'modules/queryService';

const RelatedEntitiesList = ({ match, location }) => {
    const queryConfig = queryService.getQuery(
        match,
        location,
        componentTypes.RELATED_ENTITIES_LIST
    );
    return (
        <Query query={queryConfig.query} variables={queryConfig.variables}>
            {({ loading, data }) => {
                const entityTypeText = queryConfig.metadata.entityType;
                let contents = <Loader />;
                let headerText = `Related ${entityTypeText}`;
                if (!loading && data && data.results) {
                    const results = data.results.deployments;

                    headerText = `${results.length} Related ${entityTypeText}`;
                    contents = (
                        <ul>
                            {results.map(entity => (
                                <li key={entity.id}>{entity.name}</li>
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
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

export default withRouter(RelatedEntitiesList);
