import { useEffect, useState } from 'react';

/**
 * Hook to observe the cumulative amount of space a list of elements take up
 * in a target parent element.
 *
 * @param root
 *      The parent element container to observe. An `undefined` value will use `document.body`.
 * @param observationTargets
 *      An array of descendent elements of `root` whose space should be monitored.
 * @param granularity
 *      How frequently space updates should be reported. Higher numbers will result in more frequent
 *      and more accurate updates but with a larger volume of function calls. Must be an integer >= 1.
 * @return
 *      The total `height` and `width` the target elements take up in the visible portion of the root element.
 */
export default function useLayoutSpaceObserver(
    root: HTMLElement | null | undefined,
    observationTargets: Element[],
    granularity = 10
) {
    const [usedSpace, setUsedSpace] = useState({ height: 0, width: 0 });

    useEffect(() => {
        function collectUsedSpace(intersectionEntries: IntersectionObserverEntry[]) {
            let height = 0;
            let width = 0;
            intersectionEntries.forEach(({ intersectionRect }) => {
                height += intersectionRect.height;
                width += intersectionRect.width;
            });

            if (height !== usedSpace.height || width !== usedSpace.width) {
                setUsedSpace({ height, width });
            }
        }

        const threshold = Array.from(Array(granularity + 1), (_, n) => n / granularity);
        const options = { root, rootMargin: '0px', threshold };
        const observer = new IntersectionObserver(collectUsedSpace, options);
        observationTargets.forEach((elem) => observer.observe(elem));

        return () => observer.disconnect();
    }, [root, observationTargets, granularity, usedSpace.height, usedSpace.width]);

    return usedSpace;
}
