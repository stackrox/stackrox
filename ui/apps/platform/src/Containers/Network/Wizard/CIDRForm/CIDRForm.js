import React from 'react';
import { Formik, Form, FieldArray } from 'formik';
import { PlusCircle } from 'react-feather';

import Button from 'Components/Button';
import CIDRFormRow from './CIDRFormRow';

const emptyCIDRBlockRow = {
    entity: {
        cidr: '',
        name: '',
    },
};

const CIDRForm = ({ rows }) => {
    function removeRowHandler(arrayHelpers, idx) {
        return () => {
            arrayHelpers.remove(idx);
        };
    }

    function addRowHandler(arrayHelpers) {
        return () => {
            arrayHelpers.push(emptyCIDRBlockRow);
        };
    }

    // function processCIDRBlocks(blocks) {
    //     return () => console.log(blocks);
    // }

    return (
        <Formik initialValues={rows}>
            {({ values }) => (
                <Form className="h-full">
                    <div className="h-full flex flex-col">
                        <FieldArray name="entities">
                            {(arrayHelpers) => (
                                <>
                                    <div className="flex flex-1 flex-col overflow-auto px-4 pt-4 mb-2">
                                        {values.entities &&
                                            values.entities.length > 0 &&
                                            values.entities.map((block, idx) => (
                                                <CIDRFormRow
                                                    idx={idx}
                                                    key={idx}
                                                    onRemoveRow={removeRowHandler(
                                                        arrayHelpers,
                                                        idx
                                                    )}
                                                />
                                            ))}
                                    </div>
                                    <div className="flex flex-col pb-4">
                                        <div className="flex justify-center">
                                            —
                                            <Button
                                                onClick={addRowHandler(arrayHelpers)}
                                                icon={<PlusCircle className="w-5 h-5" />}
                                                dataTestId="add-cidr-block-row-btn"
                                            />
                                            —
                                        </div>
                                    </div>
                                </>
                            )}
                        </FieldArray>
                        <div className="flex justify-center p-3 border-t border-base-300">
                            <Button
                                className="bg-success-200 border-2 border-success-500 p-2 rounded text-success-600"
                                text="Update Configuration"
                                // onClick={processCIDRBlocks(values)}
                            />
                        </div>
                    </div>
                </Form>
            )}
        </Formik>
    );
};

export default CIDRForm;
