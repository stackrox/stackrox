import React, { useEffect, useState } from 'react';
import { FormikProvider, useFormik } from 'formik';
import { Alert, Bullseye, Spinner, Modal, Button, Flex } from '@patternfly/react-core';
import * as Yup from 'yup';

import { isValidCidrBlock } from 'utils/urlUtils';
import {
    fetchCIDRBlocks,
    deleteCIDRBlock,
    postCIDRBlock,
    patchCIDRBlock,
} from 'services/NetworkService';
import useTimeout from 'hooks/useTimeout';
import { getHasDuplicateCIDRNames, getHasDuplicateCIDRAddresses } from './cidrFormUtils';
import DefaultCIDRToggle from './DefaultCIDRToggle';
import CIDRForm, { emptyCIDRBlockRow, CIDRBlockEntity, CIDRBlockEntities } from './CIDRForm';

const validationSchema = Yup.object().shape({
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

type FormCalloutType = 'success' | 'danger' | 'none';

const emptyFormCallout = {
    type: 'none' as FormCalloutType,
    message: '',
};

type CIDRFormModalProps = {
    selectedClusterId: string;
    isOpen: boolean;
    onClose: () => void;
    // updateNetworkNodes: () => void;
};

function CIDRFormModal({ selectedClusterId, isOpen, onClose }: CIDRFormModalProps) {
    const [CIDRBlocks, setCIDRBlocks] = useState<CIDRBlockEntities>({
        entities: [],
    });
    const [isLoading, setIsLoading] = useState(false);
    const [formCallout, setFormCallout] = useState<{
        type: FormCalloutType;
        message: string;
    }>(emptyFormCallout);
    const [CIDRBlocksToDelete, setCIDRBlocksToDelete] = useState<string[]>([]);

    const CIDRBlockMap: Record<string, CIDRBlockEntity> = {};
    CIDRBlocks.entities?.forEach(({ entity }) => {
        CIDRBlockMap[entity.id] = entity;
    });

    const initialValues =
        CIDRBlocks.entities.length !== 0 ? CIDRBlocks : { entities: [emptyCIDRBlockRow] };

    const formik = useFormik({
        initialValues,
        onSubmit: () => {
            updateCIDRBlocksHandler();
        },
        validationSchema,
        enableReinitialize: true,
    });
    const { dirty, isValid, submitForm, resetForm, values } = formik;

    useEffect(() => {
        if (selectedClusterId && isOpen) {
            setIsLoading(true);
            fetchCIDRBlocks(selectedClusterId)
                .then(({ response }) => {
                    const entities = response.entities.map(({ info }) => {
                        const { externalSource, id } = info;
                        const { name, cidr } = externalSource;
                        return {
                            entity: {
                                cidr,
                                name,
                                id,
                            },
                        };
                    });
                    setCIDRBlocks({ entities });
                })
                .catch(() => {
                    // TODO
                })
                .finally(() => setIsLoading(false));
        }
    }, [isOpen, selectedClusterId]);

    const [setModalCloseTimeout, cancelTimeout] = useTimeout(onCloseHandler);

    useEffect(() => {
        // When the modal is closed, cancel any callback that would close it again
        if (!isOpen) {
            cancelTimeout();
        }
    }, [isOpen, cancelTimeout]);

    function updateCIDRBlocksHandler() {
        setFormCallout(emptyFormCallout);

        const hasDuplicateCIDRNames = getHasDuplicateCIDRNames(values);
        const hasDuplicateCIDRAddresses = getHasDuplicateCIDRAddresses(values);
        if (hasDuplicateCIDRNames || hasDuplicateCIDRAddresses) {
            setFormCallout({
                type: 'danger',
                message: 'CIDR names and addresses must be unique.',
            });
            return null;
        }

        const allBlockPromises: Promise<CIDRBlockEntities | void>[] = [];
        values.entities.forEach((block) => {
            const { entity } = block;
            const { id, name, cidr } = entity;
            if (id !== '') {
                if (CIDRBlockMap[id]?.cidr !== cidr) {
                    allBlockPromises.push(
                        deleteCIDRBlock(id)
                            .then(() => {
                                postCIDRBlock(selectedClusterId, block)
                                    .then(() => {})
                                    .catch(() => {});
                            })
                            .catch(() => {})
                    );
                } else if (CIDRBlockMap[id]?.name !== name) {
                    allBlockPromises.push(patchCIDRBlock(id, name));
                }
            } else {
                allBlockPromises.push(postCIDRBlock(selectedClusterId, block));
            }
        });
        CIDRBlocksToDelete.forEach((blockId) => {
            allBlockPromises.push(deleteCIDRBlock(blockId));
        });

        Promise.all(allBlockPromises)
            .then(() => {
                setFormCallout({
                    type: 'success',
                    message:
                        'CIDR blocks have been successfully configured. This modal will now close.',
                });
                // updateNetworkNodes();
                if (isOpen) {
                    setModalCloseTimeout(2000);
                }
            })
            .catch((error) => {
                setFormCallout({
                    type: 'danger',
                    message: `There was an issue modifying and/or deleting CIDR blocks. Please check the rows below. The server responded: ${
                        (error?.message || '(no response)') as string
                    }`,
                });
                // refetch CIDR blocks and reset form
                resetForm({ values });
                setCIDRBlocksToDelete([]);
            });

        return null;
    }

    function removeRowHandler(removeRow, idx, entityId) {
        return () => {
            removeRow(idx);
            if (entityId !== '') {
                setCIDRBlocksToDelete([...CIDRBlocksToDelete, entityId]);
            }
        };
    }

    function onCloseHandler() {
        resetForm();
        setFormCallout(emptyFormCallout);
        setCIDRBlocksToDelete([]);
        onClose();
    }

    return (
        <Modal
            title="Manage CIDR blocks"
            description="Specify custom CIDR blocks and configure displaying auto-discovered CIDR blocks in the Network Graph."
            isOpen={isOpen}
            onClose={onCloseHandler}
            variant="small"
            actions={[
                <Button
                    key="confirm"
                    variant="primary"
                    onClick={submitForm}
                    isDisabled={!dirty || !isValid}
                >
                    Update configuration
                </Button>,
                <Button key="cancel" variant="link" onClick={onCloseHandler}>
                    Cancel
                </Button>,
            ]}
        >
            {isLoading && (
                <Bullseye>
                    <Spinner isSVG />
                </Bullseye>
            )}
            {CIDRBlocks.entities?.length >= 0 && !isLoading && (
                <Flex>
                    <DefaultCIDRToggle />
                    {formCallout.type !== 'none' && (
                        <Alert
                            variant={formCallout.type}
                            title={formCallout.message}
                            className="pf-u-mb-md"
                        />
                    )}
                    <Flex fullWidth={{ default: 'fullWidth' }}>
                        <FormikProvider value={formik}>
                            <CIDRForm removeRowHandler={removeRowHandler} />
                        </FormikProvider>
                    </Flex>
                </Flex>
            )}
        </Modal>
    );
}

export default CIDRFormModal;
