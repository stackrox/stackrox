import React, { ReactElement } from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Form,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    Text,
    TextArea,
    TextContent,
    TextInput,
} from '@patternfly/react-core';
import { FormikContextType } from 'formik';

import {
    EmailTemplateFormData,
    maxCustomBodyLength,
    maxCustomSubjectLength,
} from './EmailTemplate.utils';

export type EmailTemplateFormProps = {
    customBodyDefault: string;
    customSubjectDefault: string;
    formik: FormikContextType<EmailTemplateFormData>;
};

function EmailTemplateForm({
    customBodyDefault,
    customSubjectDefault,
    formik,
}: EmailTemplateFormProps): ReactElement {
    const {
        errors,
        handleBlur,
        handleChange,
        handleSubmit,
        isSubmitting,
        setFieldValue,
        touched,
        values,
    } = formik;

    const variantForBody = errors.customBody && touched.customBody ? 'error' : 'default';
    const variantForSubject = errors.customSubject && touched.customSubject ? 'error' : 'default';

    return (
        <Form className="pf-v5-u-py-lg pf-v5-u-px-lg" onSubmit={handleSubmit}>
            <FormGroup label="Email subject" fieldId="customSubject">
                <TextInput
                    id="customSubject"
                    type="text"
                    value={values.customSubject}
                    validated={variantForSubject}
                    onChange={(e) => handleChange(e)}
                    onBlur={handleBlur}
                    isDisabled={isSubmitting}
                    placeholder={customSubjectDefault}
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem variant={variantForSubject}>
                            {errors.customSubject}
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <TextContent>
                            <Text component="p" className="pf-v5-u-font-size-sm">
                                {values.customSubject.length} / {maxCustomSubjectLength} characters
                            </Text>
                        </TextContent>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            className="pf-v5-u-mt-sm"
                            variant="link"
                            isInline
                            size="sm"
                            onClick={() => setFieldValue('customSubject', '')}
                            isDisabled={values.customSubject.length === 0}
                        >
                            Reset to default
                        </Button>
                    </FlexItem>
                </Flex>
            </FormGroup>
            <FormGroup label="Email body" fieldId="customBody">
                <TextArea
                    id="customBody"
                    type="text"
                    value={values.customBody}
                    validated={variantForBody}
                    onChange={(e) => handleChange(e)}
                    onBlur={handleBlur}
                    isDisabled={isSubmitting}
                    style={{ minHeight: '250px' }}
                    placeholder={customBodyDefault}
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem variant={variantForBody}>
                            {errors.customBody}
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
                <Flex>
                    <FlexItem flex={{ default: 'flex_1' }}>
                        <TextContent>
                            <Text component="p" className="pf-v5-u-font-size-sm">
                                {values.customBody.length} / {maxCustomBodyLength} characters
                            </Text>
                        </TextContent>
                    </FlexItem>
                    <FlexItem>
                        <Button
                            className="pf-v5-u-mt-sm"
                            variant="link"
                            isInline
                            size="sm"
                            onClick={() => setFieldValue('customBody', '')}
                            isDisabled={values.customBody.length === 0}
                        >
                            Reset to default
                        </Button>
                    </FlexItem>
                </Flex>
            </FormGroup>
        </Form>
    );
}

export default EmailTemplateForm;
