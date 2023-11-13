import React, { ReactElement, useEffect, useState } from 'react';
import { Bullseye, Spinner, Button } from '@patternfly/react-core';

import { complianceEnhancedScanConfigsBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';
import {
    getScanConfig,
    ComplianceScanConfigurationStatus,
    createScanConfig,
    ComplianceScanConfiguration,
} from 'services/ComplianceEnhancedService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { BasePageAction } from 'utils/queryStringUtils';

// import ScanConfigWizard from './Wizard/ScanConfigWizard';

// type WizardScanConfigState = {
//     wizardScanConfig: ClientScanConfig;
// };

type ScanConfigPageProps = {
    // eslint-disable-next-line react/no-unused-prop-types
    hasWriteAccessForCompliance: boolean;
    pageAction?: BasePageAction;
    scanConfigId?: string;
};

function ScanConfigPage({
    // TODO: for creating new scan schedules
    // hasWriteAccessForCompliance,
    pageAction,
    scanConfigId,
}: ScanConfigPageProps): ReactElement {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [scanConfig, setScanConfig] = useState<ComplianceScanConfigurationStatus>();
    //     pageAction === 'generate' && wizardScanConfig
    //         ? getClientWizardScanConfig(wizardScanConfig)
    //         : initialScanConfig
    const [scanConfigError, setScanConfigError] = useState<ReactElement | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        setScanConfigError(null);
        if (scanConfigId) {
            // action is 'clone' or 'edit' or undefined
            setIsLoading(true);
            getScanConfig(scanConfigId)
                .then((data) => {
                    const clientWizardScanConfig = data;
                    setScanConfig(clientWizardScanConfig);
                })
                .catch((error) => {
                    // TODO: conditionally render specific or generic title string for actual status 404 or not
                    // Something like hasStatusNotFound(error) seems worthwhile in responseErrorUtils.ts file.
                    setScanConfigError(
                        <NotFoundMessage
                            title="404: We couldn't find that page"
                            message={getAxiosErrorMessage(error)}
                            actionText="Go to Scheduling main page"
                            url={complianceEnhancedScanConfigsBasePath}
                        />
                    );
                })
                .finally(() => {
                    setIsLoading(false);
                });
        }
    }, [pageAction, scanConfigId]);

    // TODO: delete
    const createFakeSchedule = async () => {
        const mockScanConfig: ComplianceScanConfiguration = {
            scanName: 'random180',
            scanConfig: {
                oneTimeScan: false,
                profiles: ['profile-5', 'profile-3'],
                scanSchedule: {
                    intervalType: 'WEEKLY',
                    hour: 20,
                    minute: 10,
                    daysOfWeek: {
                        days: [4],
                    },
                },
            },
            clusters: ['8e6fc93d-b5fb-418b-8835-1c42a512a8f6'],
        };
        await createScanConfig(mockScanConfig);
    };

    return (
        <>
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            ) : (
                scanConfigError || // TODO ROX-8487: Improve ScanConfigPage when request fails
                (pageAction ? (
                    <div>
                        <span>ScanConfigWizard goes here</span>
                        <div>
                            <Button className="pf-u-m-xl" onClick={createFakeSchedule}>
                                Create
                            </Button>
                        </div>
                    </div>
                ) : (
                    <div>ScanConfigDetail goes here</div>
                ))
            )}
        </>
    );
}

export default ScanConfigPage;
