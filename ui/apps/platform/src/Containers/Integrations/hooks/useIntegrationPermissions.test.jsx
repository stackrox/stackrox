// this test file written in JS, because mock Provider difficult to type in test context
import React from 'react';
import { renderHook } from '@testing-library/react';
import { Provider } from 'react-redux';
import { HistoryRouter as Router } from 'redux-first-history/rr6';

import configureStore from 'init/configureStore';
import useIntegrationPermissions from './useIntegrationPermissions';

const initialStoreWrite = {
    app: {
        roles: {
            userRolePermissions: {
                resourceToAccess: {
                    Integration: 'READ_WRITE_ACCESS',
                },
            },
        },
    },
};
const initialStoreRead = {
    app: {
        roles: {
            userRolePermissions: {
                resourceToAccess: {
                    Integration: 'READ_ACCESS',
                },
            },
        },
    },
};
const initialStoreNone = {
    app: {
        roles: {
            userRolePermissions: {
                resourceToAccess: {
                    Integration: 'NO_ACCESS',
                },
            },
        },
    },
};

describe('useIntegrationPermissions', () => {
    it('should return write permissions', () => {
        const { store, history } = configureStore(initialStoreWrite);

        const { result } = renderHook(() => useIntegrationPermissions(), {
            wrapper: ({ children }) => (
                <Router history={history}>
                    <Provider store={store}>{children}</Provider>
                </Router>
            ),
        });

        expect(result.current.authProviders.write).toEqual(true);
        expect(result.current.authProviders.read).toEqual(true);
        expect(result.current.notifiers.write).toEqual(true);
        expect(result.current.notifiers.read).toEqual(true);
        expect(result.current.imageIntegrations.write).toEqual(true);
        expect(result.current.imageIntegrations.read).toEqual(true);
        expect(result.current.backups.write).toEqual(true);
        expect(result.current.backups.read).toEqual(true);
    });

    it('should return read permissions', () => {
        const { store, history } = configureStore(initialStoreRead);

        const { result } = renderHook(() => useIntegrationPermissions(), {
            wrapper: ({ children }) => (
                <Router history={history}>
                    <Provider store={store}>{children}</Provider>
                </Router>
            ),
        });

        expect(result.current.authProviders.write).toEqual(false);
        expect(result.current.authProviders.read).toEqual(true);
        expect(result.current.notifiers.write).toEqual(false);
        expect(result.current.notifiers.read).toEqual(true);
        expect(result.current.imageIntegrations.write).toEqual(false);
        expect(result.current.imageIntegrations.read).toEqual(true);
        expect(result.current.backups.write).toEqual(false);
        expect(result.current.backups.read).toEqual(true);
    });

    it('should return no permissions', () => {
        const { store, history } = configureStore(initialStoreNone);

        const { result } = renderHook(() => useIntegrationPermissions(), {
            wrapper: ({ children }) => (
                <Router history={history}>
                    <Provider store={store}>{children}</Provider>
                </Router>
            ),
        });

        expect(result.current.authProviders.write).toEqual(false);
        expect(result.current.authProviders.read).toEqual(false);
        expect(result.current.notifiers.write).toEqual(false);
        expect(result.current.notifiers.read).toEqual(false);
        expect(result.current.imageIntegrations.write).toEqual(false);
        expect(result.current.imageIntegrations.read).toEqual(false);
        expect(result.current.backups.write).toEqual(false);
        expect(result.current.backups.read).toEqual(false);
    });
});
