import React from 'react';
import { renderHook } from '@testing-library/react-hooks';
import { Provider } from 'react-redux';
import { createBrowserHistory as createHistory } from 'history';

import configureStore from '../store/configureStore';
import useBranding from './useBranding';

const history = createHistory();

const rhacsMetadata = { app: { metadata: { metadata: { productBranding: 'RHACS_BRANDING' } } } };
const stackroxMetadata = {
    app: { metadata: { metadata: { productBranding: 'STACKROX_BRANDING' } } },
};
const bogusMetadata = { app: { metadata: { metadata: { productBranding: 'BOGUS' } } } };
const undefinedMetadata = { app: { metadata: { metadata: {} } } };

describe('useBranding', () => {
    it('should return Red Hat ACS Branding when RHACS_BRANDING', () => {
        const store = configureStore(rhacsMetadata, history);

        const { result } = renderHook(() => useBranding(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        expect(result.current.basePageTitle).toContain('Red Hat Advanced Cluster Security');
        expect(result.current.basePageTitle).not.toContain('StackRox');
    });

    it('should return Open Source Branding when STACKROX_BRANDING', () => {
        const store = configureStore(stackroxMetadata, history);

        const { result } = renderHook(() => useBranding(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        expect(result.current.basePageTitle).toContain('StackRox');
        expect(result.current.basePageTitle).not.toContain('Red Hat Advanced Cluster Security');
    });

    it('should return empty values when productBranding is an invalid value', () => {
        const store = configureStore(bogusMetadata, history);

        const { result } = renderHook(() => useBranding(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        expect(result.current.basePageTitle).not.toContain('StackRox');
        expect(result.current.basePageTitle).not.toContain('Red Hat Advanced Cluster Security');
    });

    it('should return empty values when productBranding is not defined', () => {
        const store = configureStore(undefinedMetadata, history);

        const { result } = renderHook(() => useBranding(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        expect(result.current.basePageTitle).not.toContain('StackRox');
        expect(result.current.basePageTitle).not.toContain('Red Hat Advanced Cluster Security');
    });
});
