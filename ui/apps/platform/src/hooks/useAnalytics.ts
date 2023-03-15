import { useCallback } from 'react';
import { useSelector } from 'react-redux';
import { Telemetry } from 'types/config.proto';

import { selectors } from 'reducers';

const useAnalytics = () => {
    const telemetry = useSelector(selectors.publicConfigTelemetrySelector);
    const { enabled: telemetryEnabled } = telemetry || ({} as Telemetry);

    const analyticsIdentity = useCallback(
        (userId: string, traits = {}): void => {
            if (telemetryEnabled) {
                window.analytics?.identify(userId, traits);
            }
        },
        [telemetryEnabled]
    );

    const analyticsPageVisit = useCallback(
        (type: string, name: string, additionalProperties = {}): void => {
            if (telemetryEnabled) {
                window.analytics?.page(type, name, additionalProperties);
            }
        },
        [telemetryEnabled]
    );

    const analyticsTrack = useCallback(
        (event: string, additionalProperties = {}): void => {
            if (telemetryEnabled) {
                window.analytics?.track(event, additionalProperties);
            }
        },
        [telemetryEnabled]
    );

    return { analyticsIdentity, analyticsPageVisit, analyticsTrack };
};

export default useAnalytics;

export const clusterCreated = 'Cluster Created';
