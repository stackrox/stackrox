import { DependencyList, EffectCallback, useEffect, useRef } from 'react';

function useEffectAfterFirstRender(eff: EffectCallback, dependencies: DependencyList): void {
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
