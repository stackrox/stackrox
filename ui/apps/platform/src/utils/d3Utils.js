import { select } from 'd3-selection';
import Raven from 'raven-js';

export const getWidth = (selector) => {
    const selectedElement = select(selector);
    if (!selectedElement) {
        Raven.captureException(new Error(`Selection for (${selector}) was not valid`));
        // TODO: re-evaluate if an exception should be thrown by this utility function to be captured up the stack
        return 0;
    }
    return parseInt(selectedElement.style('width'), 10);
};

export const getHeight = (selector) => {
    const selectedElement = select(selector);
    if (!selectedElement) {
        Raven.captureException(new Error(`Selection for (${selector}) was not valid`));
        // TODO: re-evaluate if an exception should be thrown by this utility function to be captured up the stack
        return 0;
    }
    return parseInt(selectedElement.style('height'), 10);
};
