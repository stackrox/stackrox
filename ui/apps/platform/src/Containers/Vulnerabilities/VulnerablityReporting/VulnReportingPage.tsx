import { Navigate, Route, Routes } from 'react-router-dom-v5-compat';

import type { ReportPageAction } from 'Components/Reports/reports.types';
import usePageAction from 'hooks/usePageAction';
import usePermissions from 'hooks/usePermissions';

import ImageVulnerabilityReportWizardPage from '../ImageVulnerabilityReports/Wizard/ImageVulnerabilityReportWizardPage';

import ViewVulnReportPage from './ViewVulnReport/ViewVulnReportPage';
import ConfigReportsTab from './VulnReports/ConfigReportsTab';
import ViewBasedReportsTab from './VulnReports/ViewBasedReportsTab';
import VulnReportingLayout from './VulnReports/VulnReportingLayout';

import './VulnReportingPage.css';

function VulnReportingPage() {
    const { pageAction } = usePageAction<ReportPageAction>();

    const { hasReadWriteAccess, hasReadAccess } = usePermissions();
    const hasWriteAccessForReport =
        hasReadWriteAccess('WorkflowAdministration') &&
        hasReadAccess('Image') && // for vulnerabilities
        hasReadAccess('Integration'); // for notifiers

    // TODO: Modify routing for edge cases - https://github.com/stackrox/stackrox/pull/14873#discussion_r2042672432
    return (
        <Routes>
            <Route
                element={
                    (pageAction === 'create' || pageAction === 'createFromFilters') &&
                    hasWriteAccessForReport ? (
                        <ImageVulnerabilityReportWizardPage pageAction={pageAction} />
                    ) : (
                        <VulnReportingLayout />
                    )
                }
            >
                <Route index element={<Navigate to="configuration" replace />} />
                <Route path="configuration" element={<ConfigReportsTab />} />
                <Route path="view-based" element={<ViewBasedReportsTab />} />
            </Route>
            <Route
                path="/configuration/:reportId"
                element={
                    (pageAction === 'clone' || pageAction === 'edit') && hasWriteAccessForReport ? (
                        <ImageVulnerabilityReportWizardPage pageAction={pageAction} />
                    ) : (
                        <ViewVulnReportPage />
                    )
                }
            />
        </Routes>
    );
}

export default VulnReportingPage;
