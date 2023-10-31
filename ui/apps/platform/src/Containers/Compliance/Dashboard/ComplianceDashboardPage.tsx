import React, { ReactElement, useState } from 'react';
import { useApolloClient } from '@apollo/client';

import Button from 'Components/Button';
import ExportButton from 'Components/ExportButton';
import PageHeader from 'Components/PageHeader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { resourceTypes } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { useTheme } from 'Containers/ThemeProvider';
import usePermissions from 'hooks/usePermissions';
import {
    ComplianceStandardMetadata,
    fetchComplianceStandardsSortedByName,
} from 'services/ComplianceService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import ScanButton from '../ScanButton';
import StandardsByEntity from '../widgets/StandardsByEntity';
import StandardsAcrossEntity from '../widgets/StandardsAcrossEntity';

import ManageStandardsError from './ManageStandardsError';
import ManageStandardsModal from './ManageStandardsModal';
import ComplianceDashboardTile from './ComplianceDashboardTile';

function ComplianceDashboardPage(): ReactElement {
    const { hasReadWriteAccess } = usePermissions();
    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');

    const [isFetchingStandards, setIsFetchingStandards] = useState(false);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');
    const [standards, setStandards] = useState<ComplianceStandardMetadata[]>([]);
    const [isManageStandardsModalOpen, setIsManageStandardsModalOpen] = useState(false);

    const client = useApolloClient();

    const [isExporting, setIsExporting] = useState(false);

    const { isDarkMode } = useTheme();
    const darkModeClasses = `${
        isDarkMode ? 'text-base-600 hover:bg-primary-200' : 'text-base-100 hover:bg-primary-800'
    }`;

    function clickManageStandardsButton() {
        setIsFetchingStandards(true);
        fetchComplianceStandardsSortedByName()
            .then((standardsFetched) => {
                setErrorMessageFetching('');
                setStandards(standardsFetched);
                setIsManageStandardsModalOpen(true);
            })
            .catch((error) => {
                setErrorMessageFetching(getAxiosErrorMessage(error));
                setStandards([]);
            })
            .finally(() => {
                setIsFetchingStandards(false);
            });
    }

    function onCloseManageStandardsError() {
        setErrorMessageFetching('');
    }

    function onChangeManageStandardsModal() {
        setIsManageStandardsModalOpen(false);

        /*
         * Same method as for Scan button to clear store of any cached query data,
         * so backend filters out standards in query data according to saved update
         * to hideScanResults properties.
         */
        return client.resetStore();
    }

    function onCancelManageStandardsModal() {
        setIsManageStandardsModalOpen(false);
    }

    /* eslint-disable no-nested-ternary */
    return (
        <>
            <PageHeader header="Compliance" subHeader="Dashboard">
                <div className="flex w-full justify-end">
                    <div className="flex">
                        <ComplianceDashboardTile entityType="CLUSTER" />
                        <ComplianceDashboardTile entityType="NAMESPACE" />
                        <ComplianceDashboardTile entityType="NODE" />
                        <ComplianceDashboardTile entityType="DEPLOYMENT" />
                        {hasWriteAccessForCompliance && (
                            <ScanButton
                                className={`flex items-center justify-center border-2 btn btn-base h-10 lg:min-w-32 xl:min-w-43 ${darkModeClasses}`}
                                text="Scan environment"
                                textClass="hidden lg:block"
                                textCondensed="Scan all"
                                clusterId="*"
                                standardId="*"
                            />
                        )}
                        {hasWriteAccessForCompliance && (
                            <div className="flex items-center">
                                <Button
                                    text="Manage standards"
                                    className="btn btn-base h-10 ml-2"
                                    onClick={() => {
                                        clickManageStandardsButton();
                                    }}
                                    disabled={isFetchingStandards}
                                    isLoading={isFetchingStandards}
                                />
                            </div>
                        )}
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
            </PageHeader>
            <div className="flex-1 relative p-6 xxxl:p-8 bg-base-200" id="capture-dashboard">
                <div
                    className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense pf-u-pb-lg"
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
                </div>
            </div>
            {isExporting && <BackdropExporting />}
            {errorMessageFetching ? (
                <ManageStandardsError
                    onClose={onCloseManageStandardsError}
                    errorMessage={errorMessageFetching}
                />
            ) : isManageStandardsModalOpen ? (
                <ManageStandardsModal
                    onCancel={onCancelManageStandardsModal}
                    onChange={onChangeManageStandardsModal}
                    standards={standards}
                />
            ) : null}
        </>
    );
    /* eslint-enable no-nested-ternary */
}

export default ComplianceDashboardPage;
