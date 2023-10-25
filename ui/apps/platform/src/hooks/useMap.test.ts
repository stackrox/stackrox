import { renderHook, act } from '@testing-library/react';
import useMap from './useMap';

test('useMap should test membership via reference equality', () => {
    const objA = { test: 'test' };
    const objB = { test: 'test' };

    const { result } = renderHook(() => {
        const map = useMap(new Map([[objA, 'a']]));
        return map;
    });

    expect(result.current.has(objA)).toBeTruthy();
    expect(result.current.has(objB)).toBeFalsy();
    expect(result.current.get(objA)).toBe('a');
    expect(result.current.size).toBe(1);
    expect(Array.from(result.current.values())).toEqual(['a']);

    act(() => {
        result.current.set(objA, 'b');
        result.current.set(objB, 'b');
    });

    expect(result.current.has(objA)).toBeTruthy();
    expect(result.current.has(objB)).toBeTruthy();
    expect(result.current.get(objA)).toBe('b');
    expect(result.current.get(objB)).toBe('b');
    expect(result.current.size).toBe(2);
    expect(Array.from(result.current.values())).toEqual(['b', 'b']);
});

test('useMap should correctly set, remove, and clear items', () => {
    const { result } = renderHook(() => {
        const map = useMap<string, string>();
        return map;
    });
    expect(result.current.size).toBe(0);
    expect(Array.from(result.current.values())).toEqual([]);

    act(() => {
        result.current.set('test', 'a');
    });

    expect(result.current.has('test')).toBeTruthy();
    expect(result.current.get('test')).toBe('a');
    expect(result.current.has('test-2')).toBeFalsy();
    expect(result.current.size).toBe(1);
    expect(Array.from(result.current.values())).toEqual(['a']);

    act(() => {
        result.current.set('test-2', 'b');
        result.current.remove('test');
    });

    expect(result.current.has('test')).toBeFalsy();
    expect(result.current.has('test-2')).toBeTruthy();
    expect(result.current.get('test-2')).toBe('b');

    act(() => {
        result.current.clear();
    });

    expect(result.current.has('test')).toBeFalsy();
    expect(result.current.size).toBe(0);
    expect(Array.from(result.current.values())).toEqual([]);
});
