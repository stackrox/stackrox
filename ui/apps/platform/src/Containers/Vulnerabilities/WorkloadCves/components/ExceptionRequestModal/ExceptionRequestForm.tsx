import React, { useState } from 'react';
import {
    Button,
    Flex,
    FormGroup,
    Form,
    Tabs,
    Tab,
    TextArea,
    Text,
    TabContent,
} from '@patternfly/react-core';
import { FormikHelpers, useFormik } from 'formik';

import { ScopeContext, exceptionValidationSchema, ExceptionValues } from './utils';
import ExceptionScopeField, { ALL } from './ExceptionScopeField';
import CveSelections, { CveSelectionsProps } from './CveSelections';
import ExpiryField from './ExpiryField';

function getDefaultValues(cves: string[], scopeContext: ScopeContext): ExceptionValues {
    const imageScope =
        scopeContext === 'GLOBAL'
            ? { registry: ALL, remote: ALL, tag: ALL }
            : { registry: ALL, remote: scopeContext.image.name, tag: ALL };

    return { cves, comment: '', scope: { imageScope } };
}
export type ExceptionRequestFormProps = {
    cves: CveSelectionsProps['cves'];
    scopeContext: ScopeContext;
    onSubmit: (formValues: ExceptionValues, helpers: FormikHelpers<ExceptionValues>) => void;
    onCancel: () => void;
    formHeaderText: string;
    commentFieldLabel: string;
    showExpiryField?: boolean;
};

function ExceptionRequestForm({
    cves,
    scopeContext,
    onSubmit,
    onCancel,
    formHeaderText,
    commentFieldLabel,
    showExpiryField = false,
}: ExceptionRequestFormProps) {
    const [activeKeyTab, setActiveKeyTab] = useState<string | number>('options');

    const formik = useFormik({
        initialValues: getDefaultValues(
            cves.map(({ cve }) => cve),
            scopeContext
        ),
        onSubmit,
        validationSchema: exceptionValidationSchema,
    });

    const { handleBlur, setFieldValue, touched, handleSubmit, submitForm, isSubmitting, errors } =
        formik;

    return (
        <>
            <Form
                onSubmit={handleSubmit}
                className="pf-u-display-flex pf-u-flex-direction-column"
                style={{ minHeight: 0 }}
            >
                <Tabs
                    className="pf-u-flex-shrink-0"
                    activeKey={activeKeyTab}
                    onSelect={(_, tab) => setActiveKeyTab(tab)}
                >
                    <Tab eventKey="options" title="Options" tabContentId="options" />
                    <Tab eventKey="cves" title="CVE selections" tabContentId="cves" />
                </Tabs>
                <TabContent
                    id="options"
                    className="pf-u-flex-1"
                    hidden={activeKeyTab !== 'options'}
                >
                    <Flex
                        direction={{ default: 'column' }}
                        spaceItems={{ default: 'spaceItemsLg' }}
                    >
                        <Text>{formHeaderText}</Text>
                        {showExpiryField && <ExpiryField formik={formik} />}
                        <ExceptionScopeField
                            fieldId="scope"
                            label="Scope"
                            formik={formik}
                            scopeContext={scopeContext}
                        />
                        <FormGroup fieldId="comment" label={commentFieldLabel} isRequired>
                            <TextArea
                                id="comment"
                                name="comment"
                                isRequired
                                onBlur={handleBlur('comment')}
                                onChange={(value) => setFieldValue('comment', value)}
                                validated={touched.comment && errors.comment ? 'error' : 'default'}
                            />
                        </FormGroup>
                    </Flex>
                </TabContent>
                <TabContent
                    id="cves"
                    className="pf-u-flex-1"
                    hidden={activeKeyTab !== 'cves'}
                    style={{ overflowY: 'auto' }}
                >
                    <CveSelections cves={cves} />
                </TabContent>
                <Flex>
                    <Button isLoading={isSubmitting} isDisabled={isSubmitting} onClick={submitForm}>
                        Submit request
                    </Button>
                    <Button isDisabled={isSubmitting} variant="secondary" onClick={onCancel}>
                        Cancel
                    </Button>
                </Flex>
            </Form>
        </>
    );
}

export default ExceptionRequestForm;
