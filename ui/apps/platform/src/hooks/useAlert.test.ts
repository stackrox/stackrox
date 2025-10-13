import { act, renderHook } from '@testing-library/react';

import useAlert from './useAlert';
import type { AlertObj } from './useAlert';

describe('useAlert hook', () => {
    test('useAlert should start with null', () => {
        const { result } = renderHook(() => useAlert());

        expect(result.current.alertObj).toBe(null);
        expect(typeof result.current.setAlertObj).toBe('function');
        expect(typeof result.current.clearAlertObj).toBe('function');
    });

    test('useAlert should update its stored value', () => {
        const { result } = renderHook(() => useAlert());

        const newAlert: AlertObj = {
            type: 'danger',
            title: 'The operation failed',
            children: '403 Unauthorized',
        };

        act(() => {
            result.current.setAlertObj(newAlert);
        });

        expect(result.current.alertObj).toEqual(newAlert);
    });

    test('useAlert should clear its stored value', () => {
        const { result } = renderHook(() => useAlert());

        const oldAlert: AlertObj = {
            type: 'danger',
            title: 'The operation failed',
            children: '403 Unauthorized',
        };

        act(() => {
            result.current.setAlertObj(oldAlert);
        });
        expect(result.current.alertObj).toEqual(oldAlert);

        act(() => {
            result.current.clearAlertObj();
        });
        expect(result.current.alertObj).toBe(null);
    });
});
