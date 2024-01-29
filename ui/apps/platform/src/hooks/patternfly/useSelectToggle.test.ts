import { renderHook, act } from '@testing-library/react';

import useSelectToggle from './useSelectToggle';

describe('useSelectToggle', () => {
    it('should calculate toggle states when defaulting to closed', () => {
        const { result } = renderHook(() => useSelectToggle(false));

        expect(result.current.isOpen).toEqual(false);

        act(() => {
            result.current.onToggle(result.current.isOpen);
        });
        expect(result.current.isOpen).toEqual(true);
        act(() => {
            result.current.onToggle(result.current.isOpen);
        });
        expect(result.current.isOpen).toEqual(false);

        act(() => {
            result.current.toggleSelect(false);
        });
        expect(result.current.isOpen).toEqual(false);
        act(() => {
            result.current.toggleSelect(true);
        });
        expect(result.current.isOpen).toEqual(true);

        act(() => {
            result.current.openSelect();
        });
        expect(result.current.isOpen).toEqual(true);
        // Run effect twice to test idempotency
        act(() => {
            result.current.openSelect();
        });
        expect(result.current.isOpen).toEqual(true);

        act(() => {
            result.current.closeSelect();
        });
        expect(result.current.isOpen).toEqual(false);
        // Run effect twice to test idempotency
        act(() => {
            result.current.closeSelect();
        });
        expect(result.current.isOpen).toEqual(false);
    });

    it('should calculate toggle states when defaulting to open', () => {
        const { result } = renderHook(() => useSelectToggle(true));

        expect(result.current.isOpen).toEqual(true);

        act(() => {
            result.current.onToggle(result.current.isOpen);
        });
        expect(result.current.isOpen).toEqual(false);
        act(() => {
            result.current.onToggle(result.current.isOpen);
        });
        expect(result.current.isOpen).toEqual(true);

        act(() => {
            result.current.toggleSelect(true);
        });
        expect(result.current.isOpen).toEqual(true);
        act(() => {
            result.current.toggleSelect(false);
        });
        expect(result.current.isOpen).toEqual(false);

        act(() => {
            result.current.openSelect();
        });
        expect(result.current.isOpen).toEqual(true);
        // Run effect twice to test idempotency
        act(() => {
            result.current.openSelect();
        });
        expect(result.current.isOpen).toEqual(true);

        act(() => {
            result.current.closeSelect();
        });
        expect(result.current.isOpen).toEqual(false);
        // Run effect twice to test idempotency
        act(() => {
            result.current.closeSelect();
        });
        expect(result.current.isOpen).toEqual(false);
    });
});
