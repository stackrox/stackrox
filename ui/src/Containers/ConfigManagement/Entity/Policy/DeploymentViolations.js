import React from 'react';
import PropTypes from 'prop-types';
import { entityViolationsColumns } from 'constants/listColumns';
import pluralize from 'pluralize';
import entityTypes from 'constants/entityTypes';

import NoResultsMessage from 'Components/NoResultsMessage';
import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';

const DeploymentViolations = ({ className, alerts }) => {
    if (!alerts || !alerts.length)
        return (
            <NoResultsMessage
                message="No deployments violating this policy"
                className="p-6 shadow"
                icon="info"
            />
        );
    const rows = alerts;
    const columns = entityViolationsColumns[entityTypes.DEPLOYMENT];
    return (
        <TableWidget
            header={`${rows.length} ${pluralize('Deployment', rows.length)} with Violation(s)`}
            entityType={entityTypes.DEPLOYMENT}
            columns={columns}
            rows={rows}
            idAttribute="deployment.id"
            noDataText="No Deployments with Violation(s)"
            className={className}
        />
    );
};

DeploymentViolations.propTypes = {
    className: PropTypes.string,
    alerts: PropTypes.arrayOf(PropTypes.shape({}))
};

DeploymentViolations.defaultProps = {
    className: '',
    alerts: []
};

export default DeploymentViolations;
