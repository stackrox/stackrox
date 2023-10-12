import React, { ReactElement, useEffect, useState } from 'react';
import { Bullseye, Spinner } from '@patternfly/react-core';

import { policiesBasePath } from 'routePaths';
import NotFoundMessage from 'Components/NotFoundMessage';
import PageTitle from 'Components/PageTitle';
import { getScanSchedule, ScanSchedule } from 'services/ComplianceEnhancedService';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { BasePageAction } from 'utils/queryStringUtils';

import { initialScanSchedule } from './scanschedules.utils';
// import ScanScheduleWizard from './Wizard/ScanScheduleWizard';

// type WizardScanScheduleState = {
//     wizardScanSchedule: ClientScanSchedule;
// };

type ScanSchedulePageProps = {
    // eslint-disable-next-line react/no-unused-prop-types
    hasWriteAccessForCompliance: boolean;
    pageAction?: BasePageAction;
    scanScheduleId?: string;
};

function ScanSchedulePage({
    // TODO: for creating new scan schedules
    // hasWriteAccessForCompliance,
    pageAction,
    scanScheduleId,
}: ScanSchedulePageProps): ReactElement {
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [scanSchedule, setScanSchedule] = useState<ScanSchedule>();
    //     pageAction === 'generate' && wizardScanSchedule
    //         ? getClientWizardScanSchedule(wizardScanSchedule)
    //         : initialScanSchedule
    const [scanScheduleError, setScanScheduleError] = useState<ReactElement | null>(null);
    const [isLoading, setIsLoading] = useState(false);

    useEffect(() => {
        setScanScheduleError(null);
        if (scanScheduleId) {
            // action is 'clone' or 'edit' or undefined
            setIsLoading(true);
            getScanSchedule(scanScheduleId)
                .then((data) => {
                    const clientWizardScanSchedule = data;
                    setScanSchedule(clientWizardScanSchedule);
                })
                .catch((error) => {
                    setScanSchedule(initialScanSchedule);
                    setScanScheduleError(
                        <NotFoundMessage
                            title="404: We couldn't find that page"
                            message={getAxiosErrorMessage(error)}
                            actionText="Go to Policies"
                            url={policiesBasePath}
                        />
                    );
                })
                .finally(() => {
                    setIsLoading(false);
                });
        }
    }, [pageAction, scanScheduleId]);

    return (
        <>
            <PageTitle title="ScanSchedule Management - ScanSchedule" />
            {isLoading ? (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            ) : (
                scanScheduleError || // TODO ROX-8487: Improve ScanSchedulePage when request fails
                (pageAction ? (
                    <div>ScanScheduleWizard goes here</div>
                ) : (
                    <div>ScanScheduleDetail goes here</div>
                ))
            )}
        </>
    );
}

export default ScanSchedulePage;
