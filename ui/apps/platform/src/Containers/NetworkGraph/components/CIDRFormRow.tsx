import React from 'react';
import { Field } from 'formik';
import { Button, Flex, FlexItem, FormGroup } from '@patternfly/react-core';
import { TrashIcon } from '@patternfly/react-icons';

const CIDRFormRow = ({ idx, onRemoveRow, errors, touched }) => {
    const { name: nameError, cidr: cidrError } = errors?.entity || {};
    const { name: nameTouched, cidr: cidrTouched } = touched?.entity || {};
    const showNameError = nameError && nameTouched;
    const showCidrError = cidrError && cidrTouched;
    const hasError = showNameError || showCidrError;
    let buttonClassName = '';
    if (idx === 0) {
        buttonClassName = !hasError
            ? 'pf-m-align-self-flex-end pf-u-mb-sm'
            : 'pf-m-align-self-center';
    }

    return (
        <Flex>
            <FormGroup
                label={idx === 0 ? 'CIDR name' : ''}
                isRequired
                helperTextInvalid={nameError}
                validated={showNameError ? 'error' : 'default'}
                fieldId="cidr-name"
            >
                <Field
                    name={`entities.${idx as string}.entity.name`}
                    type="text"
                    id="cidr-name"
                    placeholder="CIDR name"
                />
            </FormGroup>
            <FormGroup
                label={idx === 0 ? 'CIDR address' : ''}
                isRequired
                helperTextInvalid={cidrError}
                validated={showCidrError ? 'error' : 'default'}
            >
                <Field
                    name={`entities.${idx as string}.entity.cidr`}
                    type="text"
                    id="cidr-block-address"
                    placeholder="192.0.0.2/24"
                />
            </FormGroup>
            <FlexItem className={buttonClassName}>
                <Button
                    name={`entities.${idx as string}.entity.delete`}
                    onClick={onRemoveRow}
                    variant="plain"
                    icon={<TrashIcon />}
                />
            </FlexItem>
        </Flex>
    );
};

export default CIDRFormRow;
