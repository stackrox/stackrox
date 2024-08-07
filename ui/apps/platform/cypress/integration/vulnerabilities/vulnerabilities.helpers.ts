import { selectors as vulnSelectors } from './vulnerabilities.selectors';

/**
 * Get the count of each severity label, returns an object with the count of each severity label.
 * If a label is hidden, the count will be null.
 *
 * @param labelParentElement The parent element of the severity labels
 * @returns The count of each severity label
 */
export function getSeverityLabelCounts(labelParentElement: HTMLElement) {
    const labelValues = Array.from(
        labelParentElement.querySelectorAll('.severity-count-labels > div .pf-v5-c-label__text')
    ).map((el) => {
        const value = parseInt(el.textContent ?? '', 10);
        return Number.isNaN(value) ? null : value;
    });

    return {
        critical: labelValues[0],
        important: labelValues[1],
        moderate: labelValues[2],
        low: labelValues[3],
    };
}

export function visitEntityTab(entityType: string) {
    cy.get(vulnSelectors.entityTypeToggleItem(entityType)).click();
}
