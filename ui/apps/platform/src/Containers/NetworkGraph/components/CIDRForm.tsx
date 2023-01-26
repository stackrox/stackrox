import React from 'react';
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

function CIDRForm({ removeRowHandler }) {
    const { values, errors, touched } = useFormikContext<CIDRBlockEntities>();
    return (
        <Form>
            <FieldArray name="entities">
                {({ push, remove }) => (
                    <>
                        <Flex direction={{ default: 'column' }}>
                            {values?.entities?.map(({ entity }, idx) => (
                                <CIDRFormRow
                                    idx={idx}
                                    // eslint-disable-next-line react/no-array-index-key
                                    key={idx}
                                    onRemoveRow={removeRowHandler(remove, idx, entity.id)}
                                    errors={errors.entities?.[idx]}
                                    touched={touched.entities?.[idx]}
                                />
                            ))}
                        </Flex>
                        <FlexItem>
                            <Button
                                onClick={() => push(emptyCIDRBlockRow)}
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
