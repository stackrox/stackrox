import { renderHook, act } from '@testing-library/react-hooks';
import useLocalStorage from './useLocalStorage';

beforeEach(() => {
    window.localStorage.clear();
});

test('should safely read and write local storage', () => {
    const { result } = renderHook(() =>
        useLocalStorage('test', 'initial', (v: unknown): v is string => typeof v === 'string')
    );
    expect(result.current[0]).toBe('initial');

    act(() => {
        result.current[1]('new value');
    });
    expect(result.current[0]).toBe('new value');
});

test('should reject loading invalid values into memory when saved via raw localStorage', () => {
    // Set an invalid value in localStorage before the hook is initialized
    window.localStorage.setItem('test', '4');
    const { result } = renderHook(() =>
        useLocalStorage('test', 'initial', (v: unknown): v is string => typeof v === 'string')
    );
    // Check that the hook initializes with the initial value instead of the invalid value
    expect(result.current[0]).toBe('initial');
    expect(window.localStorage.getItem('test')).toBe('4');

    // Set a valid value via the hook
    act(() => {
        result.current[1]('new value');
    });
    expect(result.current[0]).toBe('new value');
    expect(window.localStorage.getItem('test')).toBe('"new value"');
});

test('should update in memory values when multiple hooks are used with the same key', () => {
    const predicate = (v: unknown): v is { a: string } =>
        typeof v === 'object' && v !== null && 'a' in v && typeof v.a === 'string';

    // Initialize two hooks with the same key
    const { result: result1 } = renderHook(() =>
        useLocalStorage('test-1', { a: 'init' }, predicate)
    );
    const { result: result2 } = renderHook(() =>
        useLocalStorage('test-1', { a: 'init' }, predicate)
    );

    expect(result1.current[0]).toEqual({ a: 'init' });
    expect(result1.current[0]).toEqual({ a: 'init' });

    const newVal = { a: 'new value', b: 'ignored' };

    // Only update the value in the first hook
    act(() => {
        result1.current[1](newVal);
    });

    // Both hooks should update via the storage event
    expect(result1.current[0].a).toEqual('new value');
    expect(result2.current[0].a).toEqual('new value');
});
