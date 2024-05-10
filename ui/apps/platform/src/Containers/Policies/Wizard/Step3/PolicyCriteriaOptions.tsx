import React, { useState, useEffect, useMemo } from 'react';
import {
    Button,
    Divider,
    Flex,
    Title,
    Toolbar,
    ToolbarContent,
    ToolbarItem,
    TreeView,
    TreeViewDataItem,
    TreeViewSearch,
} from '@patternfly/react-core';
import { kebabCase } from 'lodash';
import { useFormikContext } from 'formik';

import { Policy } from 'types/policy.proto';
import { Descriptor } from './policyCriteriaDescriptors';
import { getEmptyPolicyFieldCard } from '../../policies.utils';

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

type PolicyCriteriaOptionsProps = {
    descriptors: Descriptor[];
    selectedSectionIndex: number;
};

function PolicyCriteriaOptions({ descriptors, selectedSectionIndex }: PolicyCriteriaOptionsProps) {
    const [activeItems, setActiveItems] = useState<TreeViewDataItem[]>([]);
    const [allExpanded, setAllExpanded] = useState(false);
    const [filteredItems, setFilteredItems] = useState<TreeViewDataItem[]>([]);
    const [isFiltered, setIsFiltered] = useState(false);
    const { values, setFieldValue } = useFormikContext<Policy>();
    const { policyGroups } = values.policySections[selectedSectionIndex];

    const treeDataItems = useMemo(() => getPolicyFieldsAsTree([], descriptors), [descriptors]);

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

    function addPolicyFieldCardHandler(fieldCard) {
        setFieldValue(`policySections[${selectedSectionIndex as unknown as string}].policyGroups`, [
            ...policyGroups,
            fieldCard,
        ]);
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

    return (
        <Flex direction={{ default: 'column' }} spaceItems={{ default: 'spaceItemsNone' }}>
            <Title headingLevel="h2">Drag out policy fields</Title>
            <Divider component="div" className="pf-u-mb-sm pf-u-mt-md" />
            <Button variant="link" onClick={() => setAllExpanded(!allExpanded)}>
                {allExpanded && 'Collapse all'}
                {!allExpanded && 'Expand all'}
            </Button>
            <TreeView
                activeItems={activeItems}
                data={filteredItems}
                onSelect={onSelect}
                hasSelectableNodes
                variant="compactNoBackground"
                hasGuides
                allExpanded={allExpanded || isFiltered}
                toolbar={toolbar}
            />
            <Button variant="primary" onClick={addField} isDisabled={activeItems.length === 0}>
                Add policy field
            </Button>
        </Flex>
    );
}

export default PolicyCriteriaOptions;
