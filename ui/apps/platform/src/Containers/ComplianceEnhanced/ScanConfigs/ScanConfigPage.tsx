import React, { ReactElement, useEffect, useState } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { complianceEnhancedScanConfigsBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';
import PageTitle from 'Components/PageTitle';
import { getScanConfig, ScanConfig } from 'services/ComplianceEnhancedService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { BasePageAction } from 'utils/queryStringUtils';

import { initialScanConfig } from './scanConfigs.utils';
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
    const [scanConfig, setScanConfig] = useState<ScanConfig>();
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
                    setScanConfig(initialScanConfig);
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

    return (
        <>
            <PageTitle title="ScanConfig Management - ScanConfig" />
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            ) : (
                scanConfigError || // TODO ROX-8487: Improve ScanConfigPage when request fails
                (pageAction ? (
                    <div>ScanConfigWizard goes here</div>
                ) : (
                    <div>ScanConfigDetail goes here</div>
                ))
            )}
        </>
    );
}

export default ScanConfigPage;
