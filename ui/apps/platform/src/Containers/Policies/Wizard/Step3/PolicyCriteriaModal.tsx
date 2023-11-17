import React, { useState, useEffect, useMemo } from 'react';
import {
    Button,
    Flex,
    FlexItem,
    Modal,
    ModalBoxBody,
    ModalBoxFooter,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    TreeView,
    TreeViewDataItem,
    TreeViewSearch,
} from '@patternfly/react-core';
import { kebabCase } from 'lodash';

import { PolicyGroup } from 'types/policy.proto';
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
    existingGroups: PolicyGroup[];
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

function getPolicyFieldsAsTree(existingGroups, descriptors): TreeViewDataItem[] {
    const categories = getKeysByCategory(descriptors);

    const treeList = Object.keys(categories).map((category) => ({
        name: '',
        title: category,
        id: kebabCase(category),
        children: categories[category]
            .filter((child) => {
                const alreadyUsed = existingGroups.find(
                    (group) =>
                        group.fieldName.toLowerCase() === child?.name?.toLowerCase() ||
                        group.fieldName.toLowerCase() === child?.label?.toLowerCase()
                );
                return !alreadyUsed;
            })
            .map<TreeViewDataItem>((child: Descriptor) => ({
                name: child.longName,
                title: child.shortName || child.name,
                id: kebabCase(child.shortName),
            })),
    }));

    return treeList;
}

function PolicyCriteriaModal({
    addPolicyFieldCardHandler,
    descriptors,
    existingGroups,
    isModalOpen,
    onClose,
}: PolicyCriteriaModalProps) {
    const [activeItems, setActiveItems] = useState<TreeViewDataItem[]>([]);
    const [allExpanded, setAllExpanded] = useState(false);
    const [filteredItems, setFilteredItems] = useState<TreeViewDataItem[]>([]);
    const [isFiltered, setIsFiltered] = useState(false);

    const treeDataItems = useMemo(
        () => getPolicyFieldsAsTree(existingGroups, descriptors),
        [descriptors, existingGroups]
    );

    useEffect(() => {
        setFilteredItems(treeDataItems);
        setIsFiltered(false);
    }, [treeDataItems]);

    function onSearch(evt) {
        const input: string = evt.target.value;
        if (input === '') {
            setFilteredItems(treeDataItems);
            setIsFiltered(false);
        } else {
            const filtered = treeDataItems.map((item) => {
                const filteredItem = { ...item };
                if (item.children && item.children.length > 0) {
                    const filteredChildren = item.children.filter((child) => {
                        const name = typeof child.name === 'string' ? child.name : '';
                        const title = typeof child.title === 'string' ? child.title : '';
                        return (
                            name.toLowerCase().includes(input.toLowerCase()) ||
                            title.toLowerCase().includes(input.toLowerCase())
                        );
                    });

                    filteredItem.children = filteredChildren;
                }

                return filteredItem;
            });

            setFilteredItems(filtered);
            setIsFiltered(true);
        }
    }

    function onSelect(evt, treeViewItem) {
        // Ignore folders for selection
        if (treeViewItem && !treeViewItem.children) {
            setActiveItems([treeViewItem]);
        }
    }

    function addField() {
        const itemKey = activeItems[0].title;
        const itemToAdd = descriptors.find(
            (descriptor) => descriptor.shortName === itemKey || descriptor.name === itemKey
        );
        const newPolicyFieldCard = getEmptyPolicyFieldCard(itemToAdd);

        addPolicyFieldCardHandler(newPolicyFieldCard);
        close();
    }

    const toolbar = (
        <Toolbar style={{ padding: 0 }}>
            <ToolbarContent style={{ padding: 0 }}>
                <ToolbarItem widths={{ default: '100%' }}>
                    <TreeViewSearch
                        onSearch={onSearch}
                        id="input-search"
                        name="search-input"
                        aria-label="Filter policy criteria"
                    />
                </ToolbarItem>
            </ToolbarContent>
        </Toolbar>
    );

    function close() {
        setIsFiltered(false);
        setAllExpanded(false);
        setFilteredItems(treeDataItems);

        onClose();
    }

    return (
        <Modal
            title="Add policy criteria field"
            isOpen={isModalOpen}
            variant="small"
            onClose={close}
            aria-label="Add policy criteria field"
            hasNoBodyWrapper
        >
            <ModalBoxBody>
                <Flex direction={{ default: 'column' }}>
                    <FlexItem>
                        <Button variant="link" onClick={() => setAllExpanded(!allExpanded)}>
                            {allExpanded && 'Collapse all'}
                            {!allExpanded && 'Expand all'}
                        </Button>
                        <TreeView
                            activeItems={activeItems}
                            data={filteredItems}
                            onSelect={onSelect}
                            variant="compactNoBackground"
                            hasGuides
                            allExpanded={allExpanded || isFiltered}
                            toolbar={toolbar}
                        />
                    </FlexItem>
                </Flex>
            </ModalBoxBody>
            <ModalBoxFooter>
                <Button variant="primary" onClick={addField} isDisabled={activeItems.length === 0}>
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
