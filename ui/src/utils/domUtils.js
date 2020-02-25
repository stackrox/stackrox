export function adjustTooltipPosition(top, left, elementRef) {
    let [adjustedTop, adjustedLeft] = [top, left];

    if (elementRef && elementRef.current) {
        const boundingRect = elementRef.current.getBoundingClientRect();
        const windowWidth = window.innerWidth;
        const windowHeight = window.innerHeight;

        const horizontalCenter = Math.round(windowWidth / 2);
        const verticalCenter = Math.round(windowHeight / 2);

        if (boundingRect.left > horizontalCenter) {
            adjustedLeft = left - boundingRect.width;
        }
        if (boundingRect.top > verticalCenter) {
            adjustedTop = top - boundingRect.height;
        }
    }

    return [adjustedTop, adjustedLeft];
}

export default {
    adjustTooltipPosition
};
