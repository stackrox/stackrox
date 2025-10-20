import { FieldArray, useFormikContext } from 'formik';
import { PlusCircleIcon } from '@patternfly/react-icons';
import { Flex, FlexItem, Form, Button } from '@patternfly/react-core';

import CIDRFormRow from './CIDRFormRow';

export const emptyCIDRBlockRow = {
    entity: {
        cidr: '',
        name: '',
        id: '',
    },
};

export type CIDRBlockEntity = {
    cidr: string;
    name: string;
    id: string;
};

export type CIDRBlockRow = {
    entity: CIDRBlockEntity;
};

export type CIDRBlockEntities = {
    entities: CIDRBlockRow[];
};

export type CIDRFormProps = {
    removeEntity: (entityId: string) => void;
};

function CIDRForm({ removeEntity }) {
    const { values, errors, touched } = useFormikContext<CIDRBlockEntities>();
    // Replace destructuring `{ push, remove }` with `helpers.push` and `helpers.push` calls
    // because of typescript-eslint unbound-method error.
    return (
        <Form>
            <FieldArray name="entities">
                {(helpers) => (
                    <>
                        <Flex direction={{ default: 'column' }}>
                            {values?.entities?.map(({ entity }, idx) => (
                                <CIDRFormRow
                                    idx={idx}
                                    // eslint-disable-next-line react/no-array-index-key
                                    key={idx}
                                    onRemoveRow={() => {
                                        helpers.remove(idx); // from formik state
                                        removeEntity(entity.id); // for DELETE request
                                    }}
                                    errors={errors.entities?.[idx]}
                                    touched={touched.entities?.[idx]}
                                />
                            ))}
                        </Flex>
                        <FlexItem>
                            <Button
                                onClick={() => helpers.push(emptyCIDRBlockRow)}
                                icon={<PlusCircleIcon />}
                                variant="link"
                            >
                                Add CIDR block
                            </Button>
                        </FlexItem>
                    </>
                )}
            </FieldArray>
        </Form>
    );
}

export default CIDRForm;
