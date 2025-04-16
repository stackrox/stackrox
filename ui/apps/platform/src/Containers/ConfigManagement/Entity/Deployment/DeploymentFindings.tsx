import React from 'react';
import PropTypes from 'prop-types';
import FailedPoliciesAcrossDeployment from 'Containers/ConfigManagement/Entity/widgets/FailedPoliciesAcrossDeployment';
import ViolationsAcrossThisDeployment from 'Containers/Workflow/widgets/ViolationsAcrossThisDeployment';
import entityTypes from 'constants/entityTypes';

export type DeploymentFindingsProps = {
    deploymentID: string;
    entityContext?: Record<string, any>;
};

const DeploymentFindings = ({ entityContext = {}, deploymentID }: DeploymentFindingsProps) => {
    if (entityContext[entityTypes.POLICY]) {
        return (
            <ViolationsAcrossThisDeployment
                deploymentID={deploymentID}
                policyID={entityContext[entityTypes.POLICY]}
                message="No policies failed across this deployment"
            />
        );
    }
    return (
        <div className="mx-4 w-full">
            <FailedPoliciesAcrossDeployment deploymentID={deploymentID} />
        </div>
    );
};

DeploymentFindings.propTypes = {};

DeploymentFindings.defaultProps = {
    entityContext: {},
};

export default DeploymentFindings;
