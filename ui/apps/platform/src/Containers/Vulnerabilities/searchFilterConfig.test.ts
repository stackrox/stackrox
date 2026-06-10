import type {
    CompoundSearchFilterAttribute,
    DatePickerSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';

import {
    deploymentSearchFilterConfig,
    enableDateRangeConditions,
    imageCVESearchFilterConfig,
} from './searchFilterConfig';

function isDatePickerAttribute(
    attribute: CompoundSearchFilterAttribute
): attribute is DatePickerSearchFilterAttribute {
    return attribute.inputType === 'date-picker';
}

const config = [imageCVESearchFilterConfig, deploymentSearchFilterConfig];

describe('enableDateRangeConditions', () => {
    it('should enable the Between condition on date-picker attributes', () => {
        const result = enableDateRangeConditions(config);

        const datePickerAttributes = result.flatMap((entity) =>
            entity.attributes.filter(isDatePickerAttribute)
        );

        expect(datePickerAttributes.length).toBeGreaterThan(0);
        datePickerAttributes.forEach((attribute) => {
            expect(attribute.inputProps).toEqual({ enableBetweenCondition: true });
        });
    });

    it('should leave non-date-picker attributes untouched', () => {
        const result = enableDateRangeConditions(config);

        const attributePairs = result.flatMap((entity, entityIndex) =>
            entity.attributes.map((attribute, attributeIndex) => ({
                attribute,
                original: config[entityIndex].attributes[attributeIndex],
            }))
        );
        const nonDatePickerPairs = attributePairs.filter(
            ({ attribute }) => !isDatePickerAttribute(attribute)
        );

        expect(nonDatePickerPairs.length).toBeGreaterThan(0);
        nonDatePickerPairs.forEach(({ attribute, original }) => {
            expect(attribute).toBe(original);
        });
    });

    it('should not mutate the input config', () => {
        const datePickerAttributes = config.flatMap((entity) =>
            entity.attributes.filter(isDatePickerAttribute)
        );

        enableDateRangeConditions(config);

        expect(datePickerAttributes.length).toBeGreaterThan(0);
        datePickerAttributes.forEach((attribute) => {
            expect(attribute.inputProps).toBeUndefined();
        });
    });
});
