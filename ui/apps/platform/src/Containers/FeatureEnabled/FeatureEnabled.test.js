import React from 'react';
import { combineReducers, createStore } from 'redux';
import { screen } from '@testing-library/react';
import '@testing-library/jest-dom/extend-expect';

import featureFlags from 'reducers/featureFlags';
// We're using our own custom render function and not RTL's render
import renderWithRedux from 'test-utils/renderWithRedux';

import FeatureEnabled from './FeatureEnabled';

test('can render the children when the feature is enabled', () => {
    // Create a minimal store, especially to avoid createRootReducer(history) call.
    const rootReducer = combineReducers({
        app: combineReducers({ featureFlags }),
    });
    const initialState = {
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
    };
    const store = createStore(rootReducer, initialState);

    renderWithRedux(
        store,
        <FeatureEnabled featureFlag="FEATURE_TEST_1">
            {({ featureEnabled }) => featureEnabled && <div>Feature Enabled</div>}
        </FeatureEnabled>
    );
    expect(screen.getByText('Feature Enabled')).toBeDefined();
});

test("can't render the children when the feature is disabled", () => {
    // Create a minimal store, especially to avoid createRootReducer(history) call.
    const rootReducer = combineReducers({
        app: combineReducers({ featureFlags }),
    });
    const initialState = {
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
    };
    const store = createStore(rootReducer, initialState);

    renderWithRedux(
        store,
        <FeatureEnabled featureFlag="FEATURE_TEST_1">
            {({ featureEnabled }) => featureEnabled && <div>Feature Enabled</div>}
        </FeatureEnabled>
    );
    expect(screen.queryByText('Feature Enabled')).toBeNull();
});
