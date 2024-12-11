import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import { Alert } from '@patternfly/react-core';

import entityTypes from 'constants/entityTypes';
import { entityViolationsColumns } from 'constants/listColumns';

import TableWidget from 'Containers/ConfigManagement/Entity/widgets/TableWidget';

const DeploymentViolations = ({ className, alerts, entityContext }) => {
    if (!alerts || !alerts.length) {
        return (
            <Alert
                variant="success"
                isInline
                title="No deployments violating this policy"
                component="p"
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
