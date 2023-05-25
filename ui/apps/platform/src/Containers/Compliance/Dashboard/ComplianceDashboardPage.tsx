import React, { ReactElement, useEffect, useState } from 'react';

import Button from 'Components/Button';
import ExportButton from 'Components/ExportButton';
import PageHeader from 'Components/PageHeader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { resourceTypes } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { useTheme } from 'Containers/ThemeProvider';
import useFeatureFlags from 'hooks/useFeatureFlags';
import usePermissions from 'hooks/usePermissions';
import { ComplianceStandardMetadata, fetchComplianceStandards } from 'services/ComplianceService';

import ScanButton from '../ScanButton';
import StandardsByEntity from '../widgets/StandardsByEntity';
import StandardsAcrossEntity from '../widgets/StandardsAcrossEntity';
import ComplianceByStandards from '../widgets/ComplianceByStandards';

import ManageStandardsModal from './ManageStandardsModal';
import ComplianceDashboardTile from './ComplianceDashboardTile';

function ComplianceDashboardPage(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const [standards, setStandards] = useState<ComplianceStandardMetadata[]>([]);
    const [isManageStandardsModalOpen, setIsManageStandardsModalOpen] = useState(false);

    const [isExporting, setIsExporting] = useState(false);

    const { isDarkMode } = useTheme();
    const darkModeClasses = `${
        isDarkMode ? 'text-base-600 hover:bg-primary-200' : 'text-base-100 hover:bg-primary-800'
    }`;

    const hasWriteAccessForComplianceStandards = hasReadWriteAccess('Compliance'); // TODO confirm
    const isDisableComplianceStandardsEnabled = isFeatureFlagEnabled(
        'ROX_DISABLE_COMPLIANCE_STANDARDS'
    );
    const hasManageStandardsButton =
        hasWriteAccessForComplianceStandards && isDisableComplianceStandardsEnabled;

    useEffect(() => {
        fetchComplianceStandards()
            .then((standardsFetched) => {
                setStandards(standardsFetched);
            })
            .catch(() => {
                // TODO
            });
    }, []);

    function onSaveFromManageStandardsModal(standardsSaved: ComplianceStandardMetadata[]) {
        setStandards(standardsSaved);
        setIsManageStandardsModalOpen(false);
    }

    function onCancelFromManageStandardsModal() {
        setIsManageStandardsModalOpen(false);
    }

    return (
        <>
            <PageHeader header="Compliance" subHeader="Dashboard">
                <div className="flex w-full justify-end">
                    <div className="flex">
                        <ComplianceDashboardTile entityType="CLUSTER" />
                        <ComplianceDashboardTile entityType="NAMESPACE" />
                        <ComplianceDashboardTile entityType="NODE" />
                        <ComplianceDashboardTile entityType="DEPLOYMENT" />
                        <div className="ml-1 border-l-2 border-base-400 mr-3" />
                        <div className="flex items-center">
                            <div className="flex items-center">
                                <ScanButton
                                    className={`flex items-center justify-center border-2 btn btn-base h-10 uppercase lg:min-w-32 xl:min-w-43 ${darkModeClasses}`}
                                    text="Scan environment"
                                    textClass="hidden lg:block"
                                    textCondensed="Scan all"
                                    clusterId="*"
                                    standardId="*"
                                />
                            </div>
                            {hasManageStandardsButton && (
                                <div className="flex items-center">
                                    <Button
                                        text="Manage standards"
                                        className="btn btn-base h-10 ml-2"
                                        onClick={() => {
                                            setIsManageStandardsModalOpen(true);
                                        }}
                                        disabled={standards.length === 0}
                                    />
                                </div>
                            )}
                            <div className="flex items-center">
                                <ExportButton
                                    fileName="Compliance Dashboard Report"
                                    textClass="hidden lg:block"
                                    type="ALL"
                                    page={useCaseTypes.COMPLIANCE}
                                    pdfId="capture-dashboard"
                                    isExporting={isExporting}
                                    setIsExporting={setIsExporting}
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </PageHeader>
            <div className="flex-1 relative p-6 xxxl:p-8 bg-base-200" id="capture-dashboard">
                <div
                    className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense"
                    // style={{ '--min-tile-height': '160px' }}
                >
                    <StandardsAcrossEntity
                        entityType={resourceTypes.CLUSTER}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    <StandardsByEntity
                        entityType={resourceTypes.CLUSTER}
                        bodyClassName="p-4"
                        className="pdf-page"
                    />
                    <StandardsAcrossEntity
                        entityType={resourceTypes.NAMESPACE}
                        bodyClassName="px-4 pt-1"
                        className="pdf-page"
                    />
                    <StandardsAcrossEntity
                        entityType={resourceTypes.NODE}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    <ComplianceByStandards />
                </div>
            </div>
            {isExporting && <BackdropExporting />}
            {isManageStandardsModalOpen && (
                <ManageStandardsModal
                    standards={standards}
                    onSave={onSaveFromManageStandardsModal}
                    onCancel={onCancelFromManageStandardsModal}
                />
            )}
        </>
    );
}

export default ComplianceDashboardPage;
