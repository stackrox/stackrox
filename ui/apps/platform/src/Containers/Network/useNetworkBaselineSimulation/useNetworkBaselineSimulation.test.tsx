import React from 'react';
import { renderHook, act } from '@testing-library/react-hooks';
import { Provider } from 'react-redux';
import { createBrowserHistory as createHistory } from 'history';

import configureStore from 'store/configureStore';
import useNetworkBaselineSimulation from './useNetworkBaselineSimulation';

const history = createHistory();

const initialStore = {
    app: {
        network: {
            baselineSimulation: {
                isOn: false,
                options: { excludePortsAndProtocols: false },
            },
        },
    },
};

describe('useNetworkBaselineSimulation', () => {
    it('should not be in simulation mode by default', () => {
        // arrange
        const store = configureStore(initialStore, history);

        const { result } = renderHook(() => useNetworkBaselineSimulation(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        // assert
        expect(result.current.isBaselineSimulationOn).toEqual(false);
    });

    it('should be able to start a baseline simulation', () => {
        // arrange
        const store = configureStore(initialStore, history);

        const { result } = renderHook(() => useNetworkBaselineSimulation(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        // act
        act(() => {
            result.current.startBaselineSimulation({ excludePortsAndProtocols: false });
        });

        // assert
        expect(result.current.isBaselineSimulationOn).toEqual(true);
        expect(result.current.baselineSimulationOptions).toEqual({
            excludePortsAndProtocols: false,
        });
    });

    it('should be able to start a baseline simulation with options set', () => {
        // arrange
        const store = configureStore(initialStore, history);

        const { result } = renderHook(() => useNetworkBaselineSimulation(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        // act
        act(() => {
            result.current.startBaselineSimulation({ excludePortsAndProtocols: true });
        });

        // assert
        expect(result.current.isBaselineSimulationOn).toEqual(true);
        expect(result.current.baselineSimulationOptions).toEqual({
            excludePortsAndProtocols: true,
        });
    });

    it('should be able to stop a baseline simulation', () => {
        // arrange
        const store = configureStore(initialStore, history);

        const { result } = renderHook(() => useNetworkBaselineSimulation(), {
            wrapper: ({ children }) => <Provider store={store}>{children}</Provider>,
        });

        // act
        act(() => {
            result.current.startBaselineSimulation({ excludePortsAndProtocols: false });
        });
        act(() => {
            result.current.stopBaselineSimulation();
        });

        // assert
        expect(result.current.isBaselineSimulationOn).toEqual(false);
    });
});
