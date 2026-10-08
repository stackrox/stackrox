import useCaseLabels from 'messages/useCase';
import useCaseTypes from 'constants/useCaseTypes';
import usePermissions from 'hooks/usePermissions';

import DashboardLayout from 'Components/DashboardLayout';
import Header from './Header/Header';

import PolicyViolationsBySeverity from './widgets/PolicyViolationsBySeverity';
import UsersWithMostClusterAdminRoles from './widgets/UsersWithMostClusterAdminRoles';
import SecretsMostUsedAcrossDeployments from './widgets/SecretsMostUsedAcrossDeployments';

const ConfigManagementDashboardPage = () => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForPolicyViolationsBySeverity =
        hasReadAccess('Alert') && hasReadAccess('WorkflowAdministration');
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
            {hasReadAccessForUsersWithMostClusterAdminRoles && <UsersWithMostClusterAdminRoles />}
            {hasReadAccessForSecretsMostUsedAcrossDeployments && (
                <SecretsMostUsedAcrossDeployments />
            )}
        </DashboardLayout>
    );
};
export default ConfigManagementDashboardPage;
