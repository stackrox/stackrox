import { Field } from 'formik';
import {
    Button,
    Flex,
    FlexItem,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
} from '@patternfly/react-core';
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
            ? 'pf-m-align-self-flex-end pf-v5-u-mb-sm'
            : 'pf-m-align-self-center';
    }

    return (
        <Flex>
            <FormGroup label={idx === 0 ? 'CIDR name' : ''} isRequired fieldId="cidr-name">
                <Field
                    name={`entities.${idx as string}.entity.name`}
                    type="text"
                    id="cidr-name"
                    placeholder="CIDR name"
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem variant={showNameError ? 'error' : 'default'}>
                            {nameError}
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            <FormGroup label={idx === 0 ? 'CIDR address' : ''} isRequired>
                <Field
                    name={`entities.${idx as string}.entity.cidr`}
                    type="text"
                    id="cidr-block-address"
                    placeholder="192.0.0.2/24"
                />
                <FormHelperText>
                    <HelperText>
                        <HelperTextItem variant={showCidrError ? 'error' : 'default'}>
                            {cidrError}
                        </HelperTextItem>
                    </HelperText>
                </FormHelperText>
            </FormGroup>
            <FlexItem className={buttonClassName}>
                <Button
                    name={`entities.${idx as string}.entity.delete`}
                    aria-label="Delete CIDR block"
                    onClick={onRemoveRow}
                    variant="plain"
                    icon={<TrashIcon />}
                />
            </FlexItem>
        </Flex>
    );
};

export default CIDRFormRow;
