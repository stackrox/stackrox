import React, { useState } from 'react';
import useCaseLabels from 'messages/useCase';
import useCaseTypes from 'constants/useCaseTypes';
import { standardTypes } from 'constants/entityTypes';

import DashboardLayout from 'Components/DashboardLayout';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import Header from './Header/Header';

import PolicyViolationsBySeverity from './widgets/PolicyViolationsBySeverity';
import ComplianceByControls from './widgets/ComplianceByControls';
import UsersWithMostClusterAdminRoles from './widgets/UsersWithMostClusterAdminRoles';
import SecretsMostUsedAcrossDeployments from './widgets/SecretsMostUsedAcrossDeployments';

const ConfigManagementDashboardPage = () => {
    const [isExporting, setIsExporting] = useState(false);
    return (
        <>
            <DashboardLayout
                headerText={useCaseLabels[useCaseTypes.CONFIG_MANAGEMENT]}
                headerComponents={
                    <Header isExporting={isExporting} setIsExporting={setIsExporting} />
                }
            >
                <PolicyViolationsBySeverity />
                <ComplianceByControls
                    className="pdf-page"
                    standardOptions={[
                        standardTypes.CIS_Kubernetes_v1_5
                    ]}
                />
                <UsersWithMostClusterAdminRoles />
                <SecretsMostUsedAcrossDeployments />
            </DashboardLayout>
            {isExporting && <BackdropExporting />}
        </>
    );
};
export default ConfigManagementDashboardPage;
