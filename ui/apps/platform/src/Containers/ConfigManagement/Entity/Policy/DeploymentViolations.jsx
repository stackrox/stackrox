import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import entityTypes from 'constants/entityTypes';
import { entityViolationsColumns } from 'constants/listColumns';

import NoResultsMessage from 'Components/NoResultsMessage';
import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';

const DeploymentViolations = ({ className, alerts, entityContext }) => {
    if (!alerts || !alerts.length) {
        return (
            <NoResultsMessage
                message="No deployments violating this policy"
                className="p-3 shadow"
                icon="info"
            />
        );
    }
    const rows = alerts;
    const columns = entityViolationsColumns[entityTypes.DEPLOYMENT](entityContext);
    return (
        <TableWidget
            header={`${rows.length} ${pluralize('Deployment', rows.length)} with Violation(s)`}
            entityType={entityTypes.DEPLOYMENT}
            columns={columns}
            rows={rows}
            idAttribute="id"
            noDataText="No Deployments with Violation(s)"
            className={className}
            defaultSorted={[
                {
                    id: 'name',
                    desc: false,
                },
            ]}
        />
    );
};

DeploymentViolations.propTypes = {
    className: PropTypes.string,
    alerts: PropTypes.arrayOf(PropTypes.shape({})),
    entityContext: PropTypes.shape({}),
};

DeploymentViolations.defaultProps = {
    className: '',
    alerts: [],
    entityContext: {},
};

export default DeploymentViolations;
