import React, { ReactElement } from 'react';
import { TextInput, PageSection, Form, ToggleGroup, ToggleGroupItem } from '@patternfly/react-core';
import * as yup from 'yup';
import merge from 'lodash/merge';

import { ScannerV4ImageIntegration } from 'types/imageIntegration.proto';

import FormMessage from 'Components/PatternFly/FormMessage';
import FormTestButton from 'Components/PatternFly/FormTestButton';
import FormSaveButton from 'Components/PatternFly/FormSaveButton';
import FormCancelButton from 'Components/PatternFly/FormCancelButton';
import FormLabelGroup from 'Components/PatternFly/FormLabelGroup';
import useIntegrationForm from '../useIntegrationForm';
import { IntegrationFormProps } from '../integrationFormTypes';

import IntegrationFormActions from '../IntegrationFormActions';

import { categoriesUtilsForClairifyScanner } from '../../utils/integrationUtils';

const { categoriesAlternatives, getCategoriesText, matchCategoriesAlternative, validCategories } =
    categoriesUtilsForClairifyScanner;

export const validationSchema = yup.object().shape({
    name: yup.string().trim().required('An integration name is required'),
    categories: yup
        .array()
        .of(yup.string().trim().oneOf(validCategories))
        .min(1, 'Must have at least one type selected')
        .required('A category is required'),
    scannerV4: yup.object().shape({
        indexerEndpoint: yup.string().trim(),
        matcherEndpoint: yup.string().trim(),
        numConcurrentScans: yup.string().trim(),
    }),
    type: yup.string().matches(/scannerv4/),
});

export const defaultValues: ScannerV4ImageIntegration = {
    id: '',
    name: '',
    categories: ['SCANNER'],
    scannerV4: {
        numConcurrentScans: 0,
        indexerEndpoint: '',
        matcherEndpoint: '',
    },
    autogenerated: false,
    clusterId: '',
    skipTestIntegration: false,
    type: 'scannerv4',
    source: null,
};

function ScannerV4IntegrationForm({
    initialValues = null,
    isEditable = false,
}: IntegrationFormProps<ScannerV4ImageIntegration>): ReactElement {
    const formInitialValues: ScannerV4ImageIntegration = merge({}, defaultValues, initialValues);
    const {
        values,
        touched,
        errors,
        dirty,
        isValid,
        setFieldValue,
        handleBlur,
        isSubmitting,
        isTesting,
        onSave,
        onTest,
        onCancel,
        message,
    } = useIntegrationForm<ScannerV4ImageIntegration>({
        initialValues: formInitialValues,
        validationSchema,
    });

    function onChange(value, event) {
        return setFieldValue(event.target.id, value);
    }

    return (
        <>
            <PageSection variant="light" isFilled hasOverflowScroll>
                <FormMessage message={message} />
                <Form isWidthLimited>
                    <FormLabelGroup
                        label="Integration name"
                        isRequired
                        fieldId="name"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="text"
                            id="name"
                            value={values.name}
                            placeholder="(example, StackRox Scanner Integration)"
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Type"
                        isRequired
                        fieldId="categories"
                        touched={touched}
                        errors={errors}
                    >
                        <ToggleGroup id="categories" areAllGroupsDisabled>
                            {categoriesAlternatives.map((categoriesAlternative) => {
                                const [categoriesAlternativeItem0] = categoriesAlternative;
                                const text = getCategoriesText(categoriesAlternativeItem0);
                                const isSelected = matchCategoriesAlternative(
                                    categoriesAlternative,
                                    values.categories
                                );
                                return (
                                    <ToggleGroupItem
                                        key={text}
                                        text={text}
                                        isSelected={isSelected}
                                        onChange={() =>
                                            setFieldValue('categories', categoriesAlternativeItem0)
                                        }
                                    />
                                );
                            })}
                        </ToggleGroup>
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Indexer endpoint"
                        fieldId="scannerV4.indexerEndpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="scannerV4.indexerEndpoint"
                            value={values.scannerV4.indexerEndpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Matcher endpoint"
                        fieldId="scannerV4.matcherEndpoint"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            type="text"
                            id="scannerV4.matcherEndpoint"
                            value={values.scannerV4.matcherEndpoint}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                    <FormLabelGroup
                        label="Max concurrent image scans"
                        fieldId="scannerV4.numConcurrentScans"
                        helperText="0 for default"
                        touched={touched}
                        errors={errors}
                    >
                        <TextInput
                            isRequired
                            type="number"
                            id="scannerV4.numConcurrentScans"
                            value={values.scannerV4.numConcurrentScans}
                            onChange={(event, value) => onChange(value, event)}
                            onBlur={handleBlur}
                            isDisabled={!isEditable}
                        />
                    </FormLabelGroup>
                </Form>
            </PageSection>
            {isEditable && (
                <IntegrationFormActions>
                    <FormSaveButton
                        onSave={onSave}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!dirty || !isValid}
                    >
                        Save
                    </FormSaveButton>
                    <FormTestButton
                        onTest={onTest}
                        isSubmitting={isSubmitting}
                        isTesting={isTesting}
                        isDisabled={!isValid}
                    >
                        Test
                    </FormTestButton>
                    <FormCancelButton onCancel={onCancel}>Cancel</FormCancelButton>
                </IntegrationFormActions>
            )}
        </>
    );
}

export default ScannerV4IntegrationForm;
