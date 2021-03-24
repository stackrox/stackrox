import React from 'react';
import { renderHook, act } from '@testing-library/react-hooks';
import {
    BaselineSimulationProvider,
    useNetworkBaselineSimulation,
} from './baselineSimulationContext';

describe('useNetworkBaselineSimulation', () => {
    it('should not be in simulation mode by default', () => {
        // arrange
        const wrapper = ({ children }) => (
            <BaselineSimulationProvider>{children}</BaselineSimulationProvider>
        );
        const { result } = renderHook(() => useNetworkBaselineSimulation(), { wrapper });

        // assert
        expect(result.current.isBaselineSimulationOn).toEqual(false);
    });

    it('should be able to start a baseline simulation', () => {
        // arrange
        const wrapper = ({ children }) => (
            <BaselineSimulationProvider>{children}</BaselineSimulationProvider>
        );
        const { result } = renderHook(() => useNetworkBaselineSimulation(), { wrapper });

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
        const wrapper = ({ children }) => (
            <BaselineSimulationProvider>{children}</BaselineSimulationProvider>
        );
        const { result } = renderHook(() => useNetworkBaselineSimulation(), { wrapper });

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
        const wrapper = ({ children }) => (
            <BaselineSimulationProvider>{children}</BaselineSimulationProvider>
        );
        const { result } = renderHook(() => useNetworkBaselineSimulation(), { wrapper });

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
