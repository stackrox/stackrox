import React from 'react';
import PropTypes from 'prop-types';
import entityTypes from 'constants/entityTypes';

import ViolationsAcrossThisDeployment from 'Containers/Workflow/widgets/ViolationsAcrossThisDeployment';
import DeploymentViolations from './DeploymentViolations';

const PolicyFindings = ({ entityContext = {}, policyId, alerts }) => {
    if (entityContext[entityTypes.DEPLOYMENT]) {
        return (
            <ViolationsAcrossThisDeployment
                deploymentID={entityContext[entityTypes.DEPLOYMENT]}
                policyID={policyId}
                message="No deployments have failed across this policy"
            />
        );
    }
    return (
        <div className="mx-4 w-full">
            <DeploymentViolations
                className="bg-base-100"
                alerts={alerts}
                entityContext={entityContext}
            />
        </div>
    );
};

PolicyFindings.propTypes = {
    entityContext: PropTypes.shape({}),
    policyId: PropTypes.string.isRequired,
    alerts: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
};

PolicyFindings.defaultProps = {
    entityContext: {},
};

export default PolicyFindings;
