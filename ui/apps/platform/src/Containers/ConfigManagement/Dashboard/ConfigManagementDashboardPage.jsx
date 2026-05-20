import useCaseLabels from 'messages/useCase';
import useCaseTypes from 'constants/useCaseTypes';
import { standardTypes } from 'constants/entityTypes';
import usePermissions from 'hooks/usePermissions';

import DashboardLayout from 'Components/DashboardLayout';
import Header from './Header/Header';

import PolicyViolationsBySeverity from './widgets/PolicyViolationsBySeverity';
import ComplianceByControls from './widgets/ComplianceByControls';
import UsersWithMostClusterAdminRoles from './widgets/UsersWithMostClusterAdminRoles';
import SecretsMostUsedAcrossDeployments from './widgets/SecretsMostUsedAcrossDeployments';

const ConfigManagementDashboardPage = () => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForPolicyViolationsBySeverity =
        hasReadAccess('Alert') && hasReadAccess('WorkflowAdministration');
    const hasReadAccessForComplianceByControls = hasReadAccess('Compliance');
    const hasReadAccessForUsersWithMostClusterAdminRoles =
        hasReadAccess('Cluster') && hasReadAccess('K8sRoleBinding') && hasReadAccess('K8sSubject');
    const hasReadAccessForSecretsMostUsedAcrossDeployments =
        hasReadAccess('Deployment') && hasReadAccess('Secret');
    return (
        <DashboardLayout
            headerText={useCaseLabels[useCaseTypes.CONFIG_MANAGEMENT]}
            headerComponents={<Header />}
        >
            {hasReadAccessForPolicyViolationsBySeverity && <PolicyViolationsBySeverity />}
            {hasReadAccessForComplianceByControls && (
                <ComplianceByControls standardOptions={[standardTypes.CIS_Kubernetes_v1_5]} />
            )}
            {hasReadAccessForUsersWithMostClusterAdminRoles && <UsersWithMostClusterAdminRoles />}
            {hasReadAccessForSecretsMostUsedAcrossDeployments && (
                <SecretsMostUsedAcrossDeployments />
            )}
        </DashboardLayout>
    );
};
export default ConfigManagementDashboardPage;
