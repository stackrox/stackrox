import React from 'react';
import PropTypes from 'prop-types';
import { withRouter } from 'react-router-dom';
import { resourceTypes } from 'constants/entityTypes';
import componentTypes from 'constants/componentTypes';
import labels from 'messages/common';

import Query from 'Components/AppQuery';
import PageHeader from 'Components/PageHeader';
import Button from 'Components/Button';
import ScanButton from 'Containers/Compliance2/ScanButton';
import * as Icon from 'react-feather';

const handleExport = () => {
    throw new Error('"Export" is not supported yet.');
};

const subHeaderTexts = {
    [resourceTypes.NODE]: labels.resourceLabels.NODE,
    [resourceTypes.NAMESPACE]: labels.resourceLabels.NAMESPACE,
    [resourceTypes.CLUSTER]: labels.resourceLabels.CLUSTER
};

const EntityHeader = ({ params, searchComponent }) => (
    <Query params={params} componentType={componentTypes.HEADER} action="list">
        {({ loading, data }) => {
            let headerText = 'loading...';

            if (!loading && data) {
                headerText = data.results ? data.results.name : params.entityId;
            }

            return (
                <PageHeader header={headerText} subHeader={subHeaderTexts[params.entityType]}>
                    {searchComponent}
                    <div className="flex flex-1 justify-end">
                        <div className="ml-3 border-l border-base-300 mr-3" />
                        <div className="flex">
                            <div className="flex items-center mr-3">
                                <ScanButton text="Scan" clusterId={params.entityId} />
                            </div>
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

EntityHeader.propTypes = {
    searchComponent: PropTypes.element,
    params: PropTypes.shape({}).isRequired
};

EntityHeader.defaultProps = {
    searchComponent: null
};

export default withRouter(EntityHeader);
