import { renderHook, act } from '@testing-library/react';

import useToasts from './useToasts';

test('useToasts should allow addition and removal of toasts', () => {
    const { result } = renderHook(useToasts);
    expect(result.current.toasts).toHaveLength(0);

    // Test adding a single toast
    act(() => {
        const { addToast } = result.current;
        addToast('First toast');
    });
    expect(result.current.toasts).toHaveLength(1);

    // Test adding multiple toasts in succession
    act(() => {
        const { addToast } = result.current;
        addToast('Second toast');
        addToast('Third toast');
    });
    // Test reverse chronological ordering
    expect(result.current.toasts).toHaveLength(3);
    expect(result.current.toasts[2].title).toBe('First toast');
    expect(result.current.toasts[1].title).toBe('Second toast');
    expect(result.current.toasts[0].title).toBe('Third toast');
    // Test that each item has a unique key
    const toastKeys = result.current.toasts.map((t) => t.key);
    expect(toastKeys.length).toEqual(new Set(toastKeys).size);

    // Test removal of toast in the middle (e.g. via clicking the close button)
    act(() => {
        const { removeToast } = result.current;
        removeToast(result.current.toasts[1].key);
    });
    expect(result.current.toasts).toHaveLength(2);
    expect(result.current.toasts[1].title).toBe('First toast');
    expect(result.current.toasts[0].title).toBe('Third toast');

    // Test removing of multiple remaining toasts
    act(() => {
        const { removeToast } = result.current;
        result.current.toasts.map((t) => t.key).forEach(removeToast);
    });
    expect(result.current.toasts).toHaveLength(0);
});
