// system under test (SUT)
import { adjustTooltipPosition } from './domUtils';

describe('domUtils', () => {
    let windowWidth;
    let windowHeight;

    beforeEach(() => {
        windowWidth = window.innerWidth;
        windowHeight = window.innerHeight;
    });

    describe('adjustTooltipPosition', () => {
        /**
         * These test rely on the create-react-app's Jest setup, which uses JSDom to mock the DOM
         *   also, JSDom by default sets window dimensions of 1024 x 768
         *   (see https://github.com/jsdom/jsdom/blob/0cba358253fd5530af0685ac48c2535992464d06/lib/jsdom/browser/Window.js#L587-L588)
         */
        it('should return the given top and left params when element ref is falsy', () => {
            const [top, left] = [50, 75];

            const [adjustedTop, adjustedLeft] = adjustTooltipPosition(top, left, null);

            expect(adjustedTop).toEqual(top);
            expect(adjustedLeft).toEqual(left);
        });

        it('should return the given top and left params when element ref within the upper left quadrant of window', () => {
            // starts just above center, and starts just to left of center
            const [top, left] = [windowHeight / 2 - 1, windowWidth / 2 - 1];
            const boundingRect = {
                width: 128,
                height: 156,
                top,
                left
            };
            const elementRef = {
                current: {
                    getBoundingClientRect() {
                        return boundingRect;
                    }
                }
            };

            const [adjustedTop, adjustedLeft] = adjustTooltipPosition(top, left, elementRef);

            expect(adjustedTop).toEqual(top);
            expect(adjustedLeft).toEqual(left);
        });

        it('should subtract the height from the given top when element ref within the lower left quadrant of window', () => {
            // starts just below center, and starts just to left of center
            const [top, left] = [windowHeight / 2 + 1, windowWidth / 2 - 1];
            const boundingRect = {
                width: 128,
                height: 156,
                top,
                left
            };
            const elementRef = {
                current: {
                    getBoundingClientRect() {
                        return boundingRect;
                    }
                }
            };

            const [adjustedTop, adjustedLeft] = adjustTooltipPosition(top, left, elementRef);

            expect(adjustedTop).toEqual(boundingRect.top - boundingRect.height);
            expect(adjustedLeft).toEqual(left);
        });

        it('should subtract the width from the given left when element ref within the upper right quadrant of window', () => {
            // starts just below center, and starts just to left of center
            const [top, left] = [windowHeight / 2 - 1, windowWidth / 2 + 1];
            const boundingRect = {
                width: 128,
                height: 156,
                top,
                left
            };
            const elementRef = {
                current: {
                    getBoundingClientRect() {
                        return boundingRect;
                    }
                }
            };

            const [adjustedTop, adjustedLeft] = adjustTooltipPosition(top, left, elementRef);

            expect(adjustedTop).toEqual(top);
            expect(adjustedLeft).toEqual(boundingRect.left - boundingRect.width);
        });

        it('should subtract the width and height when element ref within the lower right quadrant of window', () => {
            // starts just below center, and starts just to left of center
            const [top, left] = [windowHeight / 2 + 1, windowWidth / 2 + 1];
            const boundingRect = {
                width: 128,
                height: 156,
                top,
                left
            };
            const elementRef = {
                current: {
                    getBoundingClientRect() {
                        return boundingRect;
                    }
                }
            };

            const [adjustedTop, adjustedLeft] = adjustTooltipPosition(top, left, elementRef);

            expect(adjustedTop).toEqual(boundingRect.top - boundingRect.height);
            expect(adjustedLeft).toEqual(boundingRect.left - boundingRect.width);
        });
    });
});
