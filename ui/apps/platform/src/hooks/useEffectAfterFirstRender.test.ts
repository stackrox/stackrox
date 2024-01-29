import { useState } from 'react';
import { renderHook, act } from '@testing-library/react';
import useEffectAfterFirstRender from './useEffectAfterFirstRender';

test('useEffectAfterFirstRender should mutate variable on all renders after the initial render', () => {
    let runCount = 0;
    const { result } = renderHook(() => {
        useEffectAfterFirstRender(() => {
            runCount += 1;
        });
        return useState({});
    });

    expect(runCount).toEqual(0);

    act(() => {
        const [, setObj] = result.current;
        setObj({});
    });
    expect(runCount).toEqual(1);

    act(() => {
        const [, setObj] = result.current;
        setObj({});
    });
    expect(runCount).toEqual(2);
});
