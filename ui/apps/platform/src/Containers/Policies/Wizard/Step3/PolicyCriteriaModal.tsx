import React from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Modal,
    ModalBoxBody,
    ModalBoxFooter,
    TreeView,
} from '@patternfly/react-core';

type PolicyCriteriaModalProps = {
    isModalOpen: boolean;
    onClose: () => void;
};

// TODO: remove this placeholder data
const options = [
    {
        title: 'Image registry',
        id: 'category-image-registry',
        children: [
            {
                name: 'Container registry name is',
                title: 'Image registry',
                id: 'field-image-registry',
            },
            {
                name: 'Image name is',
                title: 'Image name',
                id: 'field-image-name',
            },
            {
                name: 'Image tag is',
                title: 'Image tag',
                id: 'field-image-tag',
            },
            {
                name: 'Image signature is missing or wrong',
                title: 'Image signature',
                id: 'field-image-signature',
            },
        ],
    },
    {
        title: 'Deployment metadata',
        id: 'category-deployment metadata',
        children: [
            {
                title: 'Disallowed annotation',
                id: 'field-disallowed annotation',
            },
            {
                name: 'Required deployment label',
                title: 'Required label',
                id: 'field-required-label',
            },
            {
                name: 'Required deployment annotation',
                title: 'Required annotation',
                id: 'field-required-annotation',
            },
        ],
    },
];
// TODO end: remove this placeholder data

function PolicyCriteriaModal({ isModalOpen, onClose }: PolicyCriteriaModalProps) {
    return (
        <Modal
            title="Add policy criteria field"
            isOpen={isModalOpen}
            variant="small"
            onClose={onClose}
            aria-label="Add policy criteria field"
            hasNoBodyWrapper
        >
            <ModalBoxBody>
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>Filter by criteria name</FlexItem>
                    <FlexItem>
                        {/* eslint-disable-next-line @typescript-eslint/ban-ts-comment */}
                        {/* @ts-ignore */}
                        <TreeView data={options} variant="compact" />
                    </FlexItem>
                </Flex>
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button variant="link" onClick={onClose}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default PolicyCriteriaModal;
