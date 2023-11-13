import React, { useState } from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Modal,
    ModalBoxBody,
    ModalBoxFooter,
    TreeView,
    TreeViewDataItem,
} from '@patternfly/react-core';
import { kebabCase } from 'lodash';

import { Descriptor } from './policyCriteriaDescriptors';

import './PolicyCriteriaModal.css';

function getEmptyPolicyFieldCard(field) {
    const defaultValue = field.defaultValue !== undefined ? field.defaultValue : '';
    return {
        fieldName: field.name,
        booleanOperator: 'OR',
        values: [
            {
                value: defaultValue,
            },
        ],
        negate: false,
        field,
    };
}

type PolicyCriteriaModalProps = {
    descriptors: Descriptor[];
    isModalOpen: boolean;
    onClose: () => void;
    addPolicyFieldCardHandler: (Descriptor) => void;
};

function getKeysByCategory(keys) {
    const categories = {};
    keys.forEach((key) => {
        const { category } = key;
        if (categories[category]) {
            categories[category].push(key);
        } else {
            categories[category] = [key];
        }
    });
    return categories;
}

function PolicyCriteriaModal({
    addPolicyFieldCardHandler,
    descriptors,
    isModalOpen,
    onClose,
}: PolicyCriteriaModalProps) {
    const [activeItems, setActiveItems] = useState<TreeViewDataItem[]>([]);
    const [allExpanded, setAllExpanded] = useState(false);

    const categories = getKeysByCategory(descriptors);
    const treeList = Object.keys(categories).map((category) => ({
        name: '',
        title: category,
        id: kebabCase(category),
        children: categories[category].map((child) => ({
            name: child.longName,
            title: child.shortName,
            id: kebabCase(child.shortName),
        })),
    }));

    function onSelect(evt, treeViewItem) {
        // Ignore folders for selection
        if (treeViewItem && !treeViewItem.children) {
            setActiveItems([treeViewItem]);
        }
    }

    function addField() {
        const itemKey = activeItems[0].title;
        const itemToAdd = descriptors.find((descriptor) => descriptor.shortName === itemKey);
        const newPolicyFieldCard = getEmptyPolicyFieldCard(itemToAdd);

        addPolicyFieldCardHandler(newPolicyFieldCard);
        onClose();
    }

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
                        <Button variant="link" onClick={() => setAllExpanded(!allExpanded)}>
                            {allExpanded && 'Collapse all'}
                            {!allExpanded && 'Expand all'}
                        </Button>
                        <TreeView
                            activeItems={activeItems}
                            data={treeList}
                            onSelect={onSelect}
                            variant="compactNoBackground"
                            hasGuides
                            allExpanded={allExpanded}
                        />
                    </FlexItem>
                </Flex>
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button variant="primary" onClick={addField}>
                    Add policy field
                </Button>
                <Button variant="link" onClick={onClose}>
                    Cancel
                </Button>
            </ModalBoxFooter>
        </Modal>
    );
}

export default PolicyCriteriaModal;
