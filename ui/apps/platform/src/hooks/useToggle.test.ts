import { renderHook, act } from '@testing-library/react';

import useToggle from './useToggle';

describe('useSelectToggle', () => {
    it('should calculate toggle states when defaulting to closed', () => {
        const { result } = renderHook(() => useToggle(false));

        expect(result.current.isOn).toEqual(false);

        act(() => {
            result.current.onToggle(result.current.isOn);
        });
        expect(result.current.isOn).toEqual(true);
        act(() => {
            result.current.onToggle(result.current.isOn);
        });
        expect(result.current.isOn).toEqual(false);

        act(() => {
            result.current.toggle(false);
        });
        expect(result.current.isOn).toEqual(false);
        act(() => {
            result.current.toggle(true);
        });
        expect(result.current.isOn).toEqual(true);

        act(() => {
            result.current.toggleOn();
        });
        expect(result.current.isOn).toEqual(true);
        // Run effect twice to test idempotency
        act(() => {
            result.current.toggleOn();
        });
        expect(result.current.isOn).toEqual(true);

        act(() => {
            result.current.toggleOff();
        });
        expect(result.current.isOn).toEqual(false);
        // Run effect twice to test idempotency
        act(() => {
            result.current.toggleOff();
        });
        expect(result.current.isOn).toEqual(false);
    });

    it('should calculate toggle states when defaulting to open', () => {
        const { result } = renderHook(() => useToggle(true));

        expect(result.current.isOn).toEqual(true);

        act(() => {
            result.current.onToggle(result.current.isOn);
        });
        expect(result.current.isOn).toEqual(false);
        act(() => {
            result.current.onToggle(result.current.isOn);
        });
        expect(result.current.isOn).toEqual(true);

        act(() => {
            result.current.toggle(true);
        });
        expect(result.current.isOn).toEqual(true);
        act(() => {
            result.current.toggle(false);
        });
        expect(result.current.isOn).toEqual(false);

        act(() => {
            result.current.toggleOn();
        });
        expect(result.current.isOn).toEqual(true);
        // Run effect twice to test idempotency
        act(() => {
            result.current.toggleOn();
        });
        expect(result.current.isOn).toEqual(true);

        act(() => {
            result.current.toggleOff();
        });
        expect(result.current.isOn).toEqual(false);
        // Run effect twice to test idempotency
        act(() => {
            result.current.toggleOff();
        });
        expect(result.current.isOn).toEqual(false);
    });
});
