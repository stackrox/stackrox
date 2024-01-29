import { renderHook, act } from '@testing-library/react';

import useEntitiesByIdsCache from './useEntitiesByIdsCache';

test('should have empty array as an initial state', () => {
    const { result } = renderHook(() => useEntitiesByIdsCache());
    expect(result.current[0]).toEqual([]);
});

test('should update from initial state', () => {
    const { result } = renderHook(() => useEntitiesByIdsCache());

    const newEntities = [{ id: 1 }, { id: 2 }];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[0]).toBe(newEntities);
});

test('should update with extra entity', () => {
    const initialState = [{ id: 1 }];
    const { result } = renderHook(() => useEntitiesByIdsCache(initialState));

    const newEntities = [{ id: 1 }, { id: 2 }];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[0]).toBe(newEntities);
});

test('should update with different set of entities', () => {
    const initialState = [{ id: 1 }, { id: 2 }];
    const { result } = renderHook(() => useEntitiesByIdsCache(initialState));

    const newEntities = [{ id: 2 }, { id: 3 }];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[0]).toBe(newEntities);
});

test('should not update with same entities in different order without order respect', () => {
    const initialState = [{ id: 1 }, { id: 2 }];
    const { result } = renderHook(() =>
        useEntitiesByIdsCache(initialState, { respectOrder: false })
    );

    const newEntities = [{ id: 2 }, { id: 1 }];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[0]).toBe(initialState);
});

test('should update with same entities in different order with order respect', () => {
    const initialState = [{ id: 1 }, { id: 2 }];
    const { result } = renderHook(() => useEntitiesByIdsCache(initialState));

    const newEntities = [{ id: 2 }, { id: 1 }];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[0]).toBe(newEntities);
});

test('should respect custom ID attribute', () => {
    const initialState = [
        { id: 1, customId: 'a' },
        { id: 2, customId: 'b' },
    ];
    const { result } = renderHook(() =>
        useEntitiesByIdsCache(initialState, { idAttribute: 'customId' })
    );

    const newEntities = [
        { id: 1, customId: 'b' },
        { id: 2, customId: 'c' },
    ];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[0]).toBe(newEntities);
});

test('should not return a new object for a setter function every time', () => {
    const { result } = renderHook(() => useEntitiesByIdsCache());
    const prevSetEntities = result.current[1];

    const newEntities = [{ id: 1 }, { id: 2 }];
    act(() => {
        const setEntities = result.current[1];
        setEntities(newEntities);
    });

    expect(result.current[1]).toBe(prevSetEntities);
});
