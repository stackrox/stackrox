import React from 'react';
// We're using our own custom render function and not RTL's render
import renderWithRedux from '../../test-utils/renderWithRedux';
import '@testing-library/jest-dom/extend-expect';
import FeatureEnabled from './FeatureEnabled';

test('can render the children when the feature is enabled', () => {
    const { queryByText } = renderWithRedux(
        <FeatureEnabled featureFlag="FEATURE_TEST_1">
            {({ featureEnabled }) => featureEnabled && <div>Feature Enabled</div>}
        </FeatureEnabled>,
        {
            initialState: {
                app: {
                    featureFlags: {
                        featureFlags: [
                            {
                                name: 'FEATURE_TEST_1',
                                envVar: 'FEATURE_TEST_1',
                                enabled: true,
                            },
                        ],
                    },
                },
            },
        }
    );
    expect(queryByText('Feature Enabled')).toBeDefined();
});

test("can't render the children when the feature is disabled", () => {
    const { queryByText } = renderWithRedux(
        <FeatureEnabled featureFlag="FEATURE_TEST_1">
            {({ featureEnabled }) => featureEnabled && <div>Feature Enabled</div>}
        </FeatureEnabled>,
        {
            initialState: {
                app: {
                    featureFlags: {
                        featureFlags: [
                            {
                                name: 'FEATURE_TEST_1',
                                envVar: 'FEATURE_TEST_1',
                                enabled: false,
                            },
                        ],
                    },
                },
            },
        }
    );
    expect(queryByText('Feature Enabled')).toBeNull();
});
