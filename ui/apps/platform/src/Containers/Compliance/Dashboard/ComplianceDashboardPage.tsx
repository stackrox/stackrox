import React, { ReactElement, useState } from 'react';
import { useApolloClient } from '@apollo/client';
import { Alert } from '@patternfly/react-core';

import Button from 'Components/Button';
import ComplianceUsageDisclaimer, {
    COMPLIANCE_DISCLAIMER_KEY,
} from 'Components/ComplianceUsageDisclaimer';
import ExportButton from 'Components/ExportButton';
import PageHeader from 'Components/PageHeader';
import BackdropExporting from 'Components/PatternFly/BackdropExporting';
import { resourceTypes } from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import { useBooleanLocalStorage } from 'hooks/useLocalStorage';
import usePermissions from 'hooks/usePermissions';
import {
    ComplianceStandardMetadata,
    fetchComplianceStandardsSortedByName,
} from 'services/ComplianceService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';

import {
    AGGREGATED_RESULTS_ACROSS_ENTITY,
    AGGREGATED_RESULTS_STANDARDS_BY_ENTITY,
} from 'queries/controls';
import ScanButton from '../ScanButton';
import StandardsByEntity from '../widgets/StandardsByEntity';
import StandardsAcrossEntity from '../widgets/StandardsAcrossEntity';

import ManageStandardsError from './ManageStandardsError';
import ManageStandardsModal from './ManageStandardsModal';
import ComplianceDashboardTile, {
    CLUSTERS_COUNT,
    DEPLOYMENTS_COUNT,
    NAMESPACES_COUNT,
    NODES_COUNT,
} from './ComplianceDashboardTile';
import ComplianceScanProgress from './ComplianceScanProgress';
import { useComplianceRunStatuses } from './useComplianceRunStatuses';
import { isComplianceRouteEnabled } from '../complianceRBAC';

const queriesToRefetchOnPollingComplete = [
    CLUSTERS_COUNT,
    NODES_COUNT,
    NAMESPACES_COUNT,
    DEPLOYMENTS_COUNT,
    AGGREGATED_RESULTS_STANDARDS_BY_ENTITY(resourceTypes.CLUSTER),
    AGGREGATED_RESULTS_ACROSS_ENTITY(resourceTypes.CLUSTER),
    AGGREGATED_RESULTS_ACROSS_ENTITY(resourceTypes.NAMESPACE),
    AGGREGATED_RESULTS_ACROSS_ENTITY(resourceTypes.NODE),
];

function ComplianceDashboardPage(): ReactElement {
    const [isDisclaimerAccepted, setIsDisclaimerAccepted] = useBooleanLocalStorage(
        COMPLIANCE_DISCLAIMER_KEY,
        false
    );
    const { hasReadAccess, hasReadWriteAccess } = usePermissions();

    // Counts and widgets (with one exception) have same conditional rendering as routes.
    // Do not require for StandardsAcrossEntity cluster, so page has at least one widget.
    // Do require for StandardsByEntity cluster, because it has links to clusters.
    const isComplianceRouteEnabledForClusters = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/clusters'
    );
    const isComplianceRouteEnabledForDeployments = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/deployments'
    );
    const isComplianceRouteEnabledForNamespaces = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/namespaces'
    );
    const isComplianceRouteEnabledForNodes = isComplianceRouteEnabled(
        hasReadAccess,
        'compliance/nodes'
    );

    const hasWriteAccessForCompliance = hasReadWriteAccess('Compliance');

    const [isFetchingStandards, setIsFetchingStandards] = useState(false);
    const [errorMessageFetching, setErrorMessageFetching] = useState('');
    const [standards, setStandards] = useState<ComplianceStandardMetadata[]>([]);
    const [isManageStandardsModalOpen, setIsManageStandardsModalOpen] = useState(false);

    const client = useApolloClient();

    const [isExporting, setIsExporting] = useState(false);

    const { runs, error, restartPolling, inProgressScanDetected, isCurrentScanIncomplete } =
        useComplianceRunStatuses(queriesToRefetchOnPollingComplete);

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
                        {isComplianceRouteEnabledForClusters && (
                            <ComplianceDashboardTile entityType="CLUSTER" />
                        )}
                        {isComplianceRouteEnabledForNamespaces && (
                            <ComplianceDashboardTile entityType="NAMESPACE" />
                        )}
                        {isComplianceRouteEnabledForNodes && (
                            <ComplianceDashboardTile entityType="NODE" />
                        )}
                        {isComplianceRouteEnabledForDeployments && (
                            <ComplianceDashboardTile entityType="DEPLOYMENT" />
                        )}
                        {hasWriteAccessForCompliance && (
                            <ScanButton
                                className={`flex items-center justify-center border-2 btn btn-base h-10 lg:min-w-32 xl:min-w-43`}
                                text="Scan environment"
                                textClass="hidden lg:block"
                                textCondensed="Scan all"
                                clusterId="*"
                                standardId="*"
                                onScanTriggered={restartPolling}
                                scanInProgress={isCurrentScanIncomplete}
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
                {!isDisclaimerAccepted && (
                    <ComplianceUsageDisclaimer
                        onAccept={() => setIsDisclaimerAccepted(true)}
                        className="pf-v5-u-mb-lg"
                    />
                )}
                {(inProgressScanDetected || !!error) && (
                    <div className="pf-v5-u-pb-lg">
                        {error ? (
                            <Alert
                                variant="danger"
                                title="There was an error fetching compliance scan status, data below may be out of date"
                                component="p"
                            >
                                {getAxiosErrorMessage(error)}
                            </Alert>
                        ) : (
                            <ComplianceScanProgress runs={runs} />
                        )}
                    </div>
                )}
                <div
                    className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense pf-v5-u-pb-lg"
                    // style={{ '--min-tile-height': '160px' }}
                >
                    <StandardsAcrossEntity
                        entityType={resourceTypes.CLUSTER}
                        bodyClassName="pr-4 py-1"
                        className="pdf-page"
                    />
                    {isComplianceRouteEnabledForClusters && (
                        <StandardsByEntity
                            entityType={resourceTypes.CLUSTER}
                            bodyClassName="p-4"
                            className="pdf-page"
                        />
                    )}
                    {isComplianceRouteEnabledForNamespaces && (
                        <StandardsAcrossEntity
                            entityType={resourceTypes.NAMESPACE}
                            bodyClassName="px-4 pt-1"
                            className="pdf-page"
                        />
                    )}
                    {isComplianceRouteEnabledForNodes && (
                        <StandardsAcrossEntity
                            entityType={resourceTypes.NODE}
                            bodyClassName="pr-4 py-1"
                            className="pdf-page"
                        />
                    )}
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
