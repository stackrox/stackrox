import { useCallback } from 'react';
import { useSelector } from 'react-redux';
import { Telemetry } from 'types/config.proto';

import { selectors } from 'reducers';

// events constants
export const CLUSTER_CREATED = 'Cluster Created';
export const INVITE_USERS_MODAL_OPENED = 'Invite Users Modal Opened';
export const INVITE_USERS_SUBMITTED = 'Invite Users Submitted';
export const WATCH_IMAGE_MODAL_OPENED = 'Watch Image Modal Opened';
export const WATCH_IMAGE_SUBMITTED = 'Watch Image Submitted';

const useAnalytics = () => {
    const telemetry = useSelector(selectors.publicConfigTelemetrySelector);
    const { enabled: isTelemetryEnabled } = telemetry || ({} as Telemetry);

    const analyticsPageVisit = useCallback(
        (type: string, name: string, additionalProperties = {}): void => {
            if (isTelemetryEnabled !== false) {
                window.analytics?.page(type, name, additionalProperties);
            }
        },
        [isTelemetryEnabled]
    );

    const analyticsTrack = useCallback(
        (event: string, additionalProperties = {}): void => {
            if (isTelemetryEnabled !== false) {
                window.analytics?.track(event, additionalProperties);
            }
        },
        [isTelemetryEnabled]
    );

    return { analyticsPageVisit, analyticsTrack };
};

export default useAnalytics;
