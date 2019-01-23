import React from 'react';
import { withRouter } from 'react-router-dom';
import PropTypes from 'prop-types';
import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import * as Icon from 'react-feather';

import Query from 'Components/ThrowingQuery';
import ReactRouterPropTypes from 'react-router-prop-types';
import { resourceTypes } from 'constants/entityTypes';
import componentTypes from 'constants/componentTypes';
import queryService from 'modules/queryService';
import URLService from 'modules/URLService';
import labels from 'messages/common';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const EntityHeader = ({ match, location, searchComponent }) => {
    const subHeaderTexts = {
        [resourceTypes.NODES]: labels.resourceLabels.NODE,
        [resourceTypes.NAMESPACES]: labels.resourceLabels.NAMESPACE,
        [resourceTypes.CLUSTERS]: labels.resourceLabels.CLUSTER
    };

    const queryConfig = queryService.getQuery(match, location, componentTypes.HEADER);
    const params = new URLService(match, location).getParams();

    if (!queryConfig) return <PageHeader header="Error" subHeader="Error" />;
    return (
        <Query query={queryConfig.query} action="list" variables={queryConfig.variables}>
            {({ loading, error, data }) => {
                let headerText = 'loading...';
                if (error) {
                    // TODO: What does error state look like?
                }

                if (!loading && data) {
                    headerText = data.results ? data.results.name : params.entityId;
                }

                return (
                    <PageHeader
                        header={headerText}
                        subHeader={
                            subHeaderTexts[new URLService(match, location).getParams().entityType]
                        }
                    >
                        {searchComponent}
                        <div className="flex flex-1 justify-end">
                            <div className="ml-3 border-l border-base-300 mr-3" />
                            <div className="flex">
                                <div className="flex items-center">
                                    <Button
                                        className="btn btn-base"
                                        text="Export"
                                        icon={<Icon.FileText className="h-4 w-4 mr-3" />}
                                        onClick={handleExport}
                                    />
                                </div>
                            </div>
                        </div>
                    </PageHeader>
                );
            }}
        </Query>
    );
};
EntityHeader.propTypes = {
    searchComponent: PropTypes.element,
    match: ReactRouterPropTypes.match.isRequired,
    location: ReactRouterPropTypes.location.isRequired
};

EntityHeader.defaultProps = {
    searchComponent: null
};

export default withRouter(EntityHeader);
