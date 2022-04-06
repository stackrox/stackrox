import { DependencyList, EffectCallback, useEffect, useRef } from 'react';

// Hook that composes `useEffect` to create a variant the behaves identically after
// the first time the component renders.
function useEffectAfterFirstRender(eff: EffectCallback, dependencies?: DependencyList): void {
    const isFirstRenderRef = useRef(true);

    useEffect(() => {
        if (isFirstRenderRef.current) {
            isFirstRenderRef.current = false;
        } else {
            eff();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, dependencies);
}

export default useEffectAfterFirstRender;
