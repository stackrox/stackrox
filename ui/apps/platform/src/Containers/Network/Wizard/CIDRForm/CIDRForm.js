import React, { useState, useEffect } from 'react';
import { Formik, Form, FieldArray } from 'formik';
import { PlusCircle } from 'react-feather';

import { deleteCIDRBlock, postCIDRBlock, patchCIDRBlock } from 'services/NetworkService';
import Button from 'Components/Button';
import Message from 'Components/Message';
import CIDRFormRow from './CIDRFormRow';

const emptyCIDRBlockRow = {
    entity: {
        cidr: '',
        name: '',
    },
};

function validateName(blocks) {
    return (value) => {
        let errorMessage;
        if (!value) {
            errorMessage = 'CIDR name is required.';
            return errorMessage;
        }
        const numDuplicate = blocks.reduce(
            (acc, { entity }) => (entity.name === value ? 1 + acc : acc),
            0
        );
        if (numDuplicate > 1) {
            errorMessage = 'This field contains a duplicate CIDR name.';
        }
        return errorMessage;
    };
}

function validateAddress(blocks) {
    return (value) => {
        let errorMessage;
        if (!value) {
            errorMessage = 'CIDR address is required.';
            return errorMessage;
        }
        if (!/^([0-9]{1,3}\.){3}[0-9]{1,3}(\/([0-9]|[1-2][0-9]|3[0-2]))?$/i.test(value)) {
            errorMessage = 'CIDR address format is invalid.';
            return errorMessage;
        }
        const numDuplicate = blocks.reduce(
            (acc, { entity }) => (entity.cidr === value ? 1 + acc : acc),
            0
        );
        if (numDuplicate > 1) {
            errorMessage = 'This field contains a duplicate CIDR address.';
        }
        return errorMessage;
    };
}

const CIDRForm = ({ rows, clusterId, updateNetworkNodes, onClose }) => {
    const [hasErrors, setHasErrors] = useState();
    let blocksToRemove = [];
    const CIDRBlockMap = {};

    function setCIDRBlockMap() {
        return rows?.entities?.forEach(({ entity }) => {
            CIDRBlockMap[entity.id] = entity;
        });
    }

    function removeRowHandler(removeRow, idx, entity) {
        return () => {
            removeRow(idx);
            if (entity.id) {
                blocksToRemove.push(entity.id);
            }
        };
    }

    function addRowHandler(addRow) {
        return () => {
            addRow(emptyCIDRBlockRow);
        };
    }

    function updateCIDRBlocksHandler(values, resetForm) {
        return () => {
            const allBlockPromises = [];
            values.entities.forEach((block) => {
                const { entity } = block;
                const { id, name, cidr } = entity;
                if (id) {
                    if (CIDRBlockMap[id]?.cidr !== cidr) {
                        allBlockPromises.push(
                            deleteCIDRBlock(id).then(() => {
                                postCIDRBlock(clusterId, block);
                            })
                        );
                    } else if (CIDRBlockMap[id]?.name !== name) {
                        allBlockPromises.push(patchCIDRBlock(id, name));
                    }
                } else {
                    allBlockPromises.push(postCIDRBlock(clusterId, block));
                }
            });
            blocksToRemove.forEach((blockId) => {
                allBlockPromises.push(deleteCIDRBlock(blockId));
            });

            Promise.all(allBlockPromises)
                .then(() => {
                    setHasErrors(false);
                    updateNetworkNodes();
                    setTimeout(onClose, 2000);
                })
                .catch(() => {
                    setHasErrors(true);
                    // refetch CIDR blocks and reset form ?
                    resetForm({ values });
                    blocksToRemove = [];
                });
        };
    }

    function getInitialValues() {
        return rows.entities.length !== 0 ? rows : { entities: [emptyCIDRBlockRow] };
    }

    useEffect(setCIDRBlockMap, [rows]);

    return (
        <div className="flex flex-1 flex-col">
            {!!hasErrors && (
                <Message
                    type="error"
                    message="There was an issue modifying and/or deleting CIDR blocks. Please check the rows below."
                />
            )}
            {!hasErrors && hasErrors !== undefined && (
                <Message
                    type="info"
                    message="CIDR blocks have been successfully configured. This panel will now close."
                />
            )}
            <Formik initialValues={getInitialValues()}>
                {({ values, dirty, errors, touched, isValid, resetForm }) => (
                    <Form className="h-full">
                        <div className="h-full flex flex-col">
                            <FieldArray name="entities">
                                {({ push, remove }) => (
                                    <>
                                        <div className="flex flex-1 flex-col overflow-auto px-4 pt-4 mb-2">
                                            {values?.entities?.map(({ entity }, idx) => (
                                                <CIDRFormRow
                                                    idx={idx}
                                                    key={idx}
                                                    onRemoveRow={removeRowHandler(
                                                        remove,
                                                        idx,
                                                        entity
                                                    )}
                                                    validateName={validateName(
                                                        values.entities,
                                                        idx
                                                    )}
                                                    validateAddress={validateAddress(
                                                        values.entities,
                                                        idx
                                                    )}
                                                    errors={errors}
                                                    touched={touched}
                                                />
                                            ))}
                                        </div>
                                        <div className="flex flex-col pb-4">
                                            <div className="flex justify-center">
                                                —
                                                <Button
                                                    onClick={addRowHandler(push)}
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
                                    disabled={!dirty || !isValid}
                                    onClick={updateCIDRBlocksHandler(values, resetForm)}
                                />
                            </div>
                        </div>
                    </Form>
                )}
            </Formik>
        </div>
    );
};

export default CIDRForm;
