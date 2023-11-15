import { renderHook, act } from '@testing-library/react';
import useTimeout from './useTimeout';

test('should call the passed callback after the specified delay', async () => {
    jest.useFakeTimers();
    const callback = jest.fn();
    const { result } = renderHook(() => useTimeout(callback));

    act(() => {
        const [startTimeout] = result.current;
        startTimeout(1000);
    });

    expect(callback).not.toHaveBeenCalled();
    jest.advanceTimersByTime(500);
    expect(callback).not.toHaveBeenCalled();
    jest.advanceTimersByTime(500);
    expect(callback).toHaveBeenCalled();
});

test('should call the passed callback after the specified delay with the specified arguments', async () => {
    jest.useFakeTimers();
    const callback = jest.fn((a: string, b: string) => a + b);
    const { result } = renderHook(() => useTimeout(callback));

    act(() => {
        const [startTimeout] = result.current;
        startTimeout(1000, 'arg1', 'arg2');
    });

    expect(callback).not.toHaveBeenCalled();
    jest.advanceTimersByTime(1000);
    expect(callback).toHaveBeenCalledWith('arg1', 'arg2');
});

test('should cancel the timeout when the component is unmounted', async () => {
    jest.useFakeTimers();
    const callback = jest.fn();
    const { result, unmount } = renderHook(() => useTimeout(callback));

    act(() => {
        const [startTimeout] = result.current;
        startTimeout(1000);
    });

    expect(callback).not.toHaveBeenCalled();
    unmount();
    jest.advanceTimersByTime(1000);
    expect(callback).not.toHaveBeenCalled();
});

test('should cancel the timeout when the returned cleanup function is called', async () => {
    jest.useFakeTimers();
    const callback = jest.fn();
    const { result } = renderHook(() => useTimeout(callback));

    act(() => {
        const [startTimeout, cancelTimeout] = result.current;
        startTimeout(1000);
        jest.advanceTimersByTime(500);
        cancelTimeout();
    });

    expect(callback).not.toHaveBeenCalled();
    jest.advanceTimersByTime(500);
    expect(callback).not.toHaveBeenCalled();
});
