import type { FeatureFlag } from 'types/featureFlagService.proto';

import axios from './instance';

const url = '/v1/featureflags';

/**
 * Fetches the list of feature flags and their current values from the backend.
 */
export function fetchFeatureFlags(): Promise<{ featureFlags: FeatureFlag[] }> {
    return axios.get<{ featureFlags: FeatureFlag[] }>(url).then(({ data }) => data);
}
