import React, { useState } from 'react';
import useCaseLabels from 'messages/useCase';
import useCaseTypes from 'constants/useCaseTypes';
import { standardTypes } from 'constants/entityTypes';
import usePermissions from 'hooks/usePermissions';

import DashboardLayout from 'Components/DashboardLayout';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import Header from './Header/Header';

import PolicyViolationsBySeverity from './widgets/PolicyViolationsBySeverity';
import ComplianceByControls from './widgets/ComplianceByControls';
import UsersWithMostClusterAdminRoles from './widgets/UsersWithMostClusterAdminRoles';
import SecretsMostUsedAcrossDeployments from './widgets/SecretsMostUsedAcrossDeployments';

const ConfigManagementDashboardPage = () => {
    const [isExporting, setIsExporting] = useState(false);
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForPolicyViolationsBySeverity =
        hasReadAccess('Alert') && hasReadAccess('WorkflowAdministration');
    const hasReadAccessForComplianceByControls = hasReadAccess('Compliance');
    const hasReadAccessForUsersWithMostClusterAdminRoles =
        hasReadAccess('Cluster') && hasReadAccess('K8sRoleBinding') && hasReadAccess('K8sSubject');
    const hasReadAccessForSecretsMostUsedAcrossDeployments =
        hasReadAccess('Deployment') && hasReadAccess('Secret');
    return (
        <>
            <DashboardLayout
                headerText={useCaseLabels[useCaseTypes.CONFIG_MANAGEMENT]}
                headerComponents={
                    <Header isExporting={isExporting} setIsExporting={setIsExporting} />
                }
            >
                {hasReadAccessForPolicyViolationsBySeverity && <PolicyViolationsBySeverity />}
                {hasReadAccessForComplianceByControls && (
                    <ComplianceByControls
                        className="pdf-page"
                        standardOptions={[standardTypes.CIS_Kubernetes_v1_5]}
                    />
                )}
                {hasReadAccessForUsersWithMostClusterAdminRoles && (
                    <UsersWithMostClusterAdminRoles />
                )}
                {hasReadAccessForSecretsMostUsedAcrossDeployments && (
                    <SecretsMostUsedAcrossDeployments />
                )}
            </DashboardLayout>
            {isExporting && <BackdropExporting />}
        </>
    );
};
export default ConfigManagementDashboardPage;
