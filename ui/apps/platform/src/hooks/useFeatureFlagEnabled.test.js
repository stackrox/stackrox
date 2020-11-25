import React from 'react';
import { renderHook } from '@testing-library/react-hooks';
import { Provider } from 'react-redux';
import { createBrowserHistory as createHistory } from 'history';

import configureStore from '../store/configureStore';
import useFeatureFlagEnabled from './useFeatureFlagEnabled';

const history = createHistory();

const initialStore = {
    app: {
        featureFlags: {
            featureFlags: [
                { name: 'ENABLED', envVar: 'ROX_FEATURE_ENABLED', enabled: true },
                { name: 'DISABLED', envVar: 'ROX_FEATURE_DISABLED', enabled: false },
            ],
        },
    },
};

describe('useFeatureFlagEnabled', () => {
    it('should show the feature flag enabled', () => {
        const store = configureStore(initialStore, history);

        const { result } = renderHook(() => useFeatureFlagEnabled('ROX_FEATURE_ENABLED'), {
            // eslint-disable-next-line react/display-name
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        expect(result.current).toEqual(true);
    });

    it('should show the feature flag disabled', () => {
        const store = configureStore(initialStore, history);

        const { result } = renderHook(() => useFeatureFlagEnabled('ROX_FEATURE_DISABLED'), {
            // eslint-disable-next-line react/display-name
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        expect(result.current).toEqual(false);
    });

    describe('sad path suite', () => {
        // Note: we have to do the beforeEach/afterEach, in order to prevent of global console.error check from throwing a false positive
        const OLD_ENV = process.env;

        let spy; // for checking the console.warn call

        beforeEach(() => {
            jest.resetModules(); // this is important - it clears the cache
            process.env = { ...OLD_ENV };
            delete process.env.NODE_ENV;

            spy = jest.spyOn(console, 'error').mockImplementation();
        });

        afterEach(() => {
            process.env = OLD_ENV;

            spy.mockRestore();
        });

        it('should throw when an unknown feature flag is given', () => {
            const unknownFeature = 'ROX_UNKNOWN_FEATURE';

            expect(() => {
                const store = configureStore(initialStore, history);

                const { result } = renderHook(() => useFeatureFlagEnabled(unknownFeature), {
                    // eslint-disable-next-line react/display-name
                    wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
                });

                // The `renderHook` test helper catches the error itself
                //   see: https://react-hooks-testing-library.com/reference/api#renderhook-result
                // We could check the error here, but I prefer the standard Jest idiom to relying on this behavior
                if (result.error) {
                    throw new Error(result.error);
                }
            }).toThrow(`Feature Flag (${unknownFeature}) is not a valid feature flag`);
        });
    });
});
