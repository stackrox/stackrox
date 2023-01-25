import React, { useState, useEffect } from 'react';
import { Formik, Form, FieldArray } from 'formik';
import { PlusCircle } from 'react-feather';
import * as Yup from 'yup';
import { Message } from '@stackrox/ui-components';

import { deleteCIDRBlock, postCIDRBlock, patchCIDRBlock } from 'services/NetworkService';
import Button from 'Components/Button';
import { isValidCidrBlock } from 'utils/urlUtils';
import { getHasDuplicateCIDRNames, getHasDuplicateCIDRAddresses } from './cidrFormUtils';
import CIDRFormRow from './CIDRFormRow';

const emptyCIDRBlockRow = {
    entity: {
        cidr: '',
        name: '',
    },
};

const validateSchema = Yup.object().shape({
    entities: Yup.array().of(
        Yup.object().shape({
            entity: Yup.object().shape({
                name: Yup.string().trim().required('CIDR name is required.'),
                cidr: Yup.string()
                    .trim()
                    .test('valid-cidr-format', 'CIDR address format is invalid.', (value) => {
                        return isValidCidrBlock(value);
                    })
                    .required('CIDR address is required.'),
            }),
        })
    ),
});

const CIDRForm = ({ rows, clusterId, updateNetworkNodes, onClose }) => {
    const [formCallout, setFormCallout] = useState();
    const [CIDRBlockMap, setCIDRBlockMap] = useState({});
    let blocksToRemove = [];

    function updateCIDRBlockMap() {
        const newMap = rows?.entities?.reduce((acc, { entity }) => {
            acc[entity.id] = entity;
            return acc;
        }, {});

        setCIDRBlockMap(newMap);
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
            setFormCallout(null);

            const hasDuplicateCIDRNames = getHasDuplicateCIDRNames(values);
            const hasDuplicateCIDRAddresses = getHasDuplicateCIDRAddresses(values);
            if (hasDuplicateCIDRNames || hasDuplicateCIDRAddresses) {
                setFormCallout({
                    type: 'error',
                    message: 'CIDR names and addresses must be unique.',
                });
                return null;
            }

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
                    setFormCallout({
                        type: 'success',
                        message:
                            'CIDR blocks have been successfully configured. This panel will now close.',
                    });
                    updateNetworkNodes();
                    setTimeout(onClose, 2000);
                })
                .catch((error) => {
                    setFormCallout({
                        type: 'error',
                        message: `There was an issue modifying and/or deleting CIDR blocks. Please check the rows below. The server responded: ${
                            error?.message || '(no response)'
                        }`,
                    });
                    // refetch CIDR blocks and reset form ?
                    resetForm({ values });
                    blocksToRemove = [];
                });

            return null;
        };
    }

    function getInitialValues() {
        return rows.entities.length !== 0 ? rows : { entities: [emptyCIDRBlockRow] };
    }

    useEffect(updateCIDRBlockMap, [rows]);

    return (
        <div className="flex flex-1 flex-col">
            {formCallout && (
                <div className="mx-4">
                    <Message type={formCallout.type}>{formCallout.message}</Message>
                </div>
            )}
            <Formik initialValues={getInitialValues()} validationSchema={validateSchema}>
                {({ values, dirty, errors, touched, isValid, resetForm }) => {
                    return (
                        <Form className="h-full">
                            <div className="h-full flex flex-col">
                                <FieldArray name="entities">
                                    {({ push, remove }) => (
                                        <>
                                            <div className="flex flex-1 flex-col overflow-auto px-4 pt-4 mb-2">
                                                {values?.entities?.map(({ entity }, idx) => (
                                                    <CIDRFormRow
                                                        idx={idx}
                                                        // eslint-disable-next-line react/no-array-index-key
                                                        key={idx}
                                                        onRemoveRow={removeRowHandler(
                                                            remove,
                                                            idx,
                                                            entity
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
                    );
                }}
            </Formik>
        </div>
    );
};

export default CIDRForm;
