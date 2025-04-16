import React from 'react';
import FailedPoliciesAcrossDeployment from 'Containers/ConfigManagement/Entity/widgets/FailedPoliciesAcrossDeployment';
import ViolationsAcrossThisDeployment from 'Containers/Workflow/widgets/ViolationsAcrossThisDeployment';
import entityTypes from 'constants/entityTypes';

export type DeploymentFindingsProps = {
    deploymentID: string;
    entityContext?: Record<string, any>;
};

function DeploymentFindings({ entityContext = {}, deploymentID }: DeploymentFindingsProps) {
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
}

export default DeploymentFindings;
