import React, { ReactElement, useContext, useState } from 'react';
import { Card, InputGroup, Modal, ModalVariant, TextInput } from '@patternfly/react-core';
import { TableComposable, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';

import workflowStateContext from 'Containers/workflowStateContext';
import { EmbeddedImageScanComponent } from '../imageVulnerabilities.graphql';

export type AffectedComponentsModalProps = {
    isOpen: boolean;
    components: EmbeddedImageScanComponent[];
    onClose: () => void;
};

function AffectedComponentsModal({
    isOpen,
    components,
    onClose,
}: AffectedComponentsModalProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    const [inputValue, setInputValue] = useState('');

    function onInputValueChange(value) {
        setInputValue(value);
    }

    function onCloseHandler() {
        setInputValue('');
        onClose();
    }

    const filteredComponents = components.filter((component) => {
        return component.name.includes(inputValue);
    });

    return (
        <Modal
            variant={ModalVariant.small}
            title="Affected Components"
            isOpen={isOpen}
            onClose={onCloseHandler}
        >
            <InputGroup className="pf-u-mt-md">
                <TextInput
                    name="componentsFilter"
                    id="componentsFilter"
                    type="text"
                    aria-label="Filter components"
                    placeholder="Filter components"
                    value={inputValue}
                    onChange={onInputValueChange}
                />
            </InputGroup>
            <Card isFlat className="pf-u-mt-lg">
                <TableComposable aria-label="Affected Components Table" variant="compact" borders>
                    <Thead>
                        <Tr>
                            <Th>Component</Th>
                            <Th>Version</Th>
                            <Th>Fixed in</Th>
                        </Tr>
                    </Thead>
                    <Tbody>
                        {filteredComponents.map((component) => {
                            const componentURL = workflowState
                                .pushList('COMPONENT')
                                .pushListItem(component.id)
                                .toUrl();

                            return (
                                <Tr key={component.name}>
                                    <Td dataLabel="Component">
                                        <a href={componentURL} target="_blank" rel="noreferrer">
                                            {component.name}
                                        </a>
                                    </Td>
                                    <Td dataLabel="Version">{component.version}</Td>
                                    <Td dataLabel="Fixed in">{component.fixedIn || '-'}</Td>
                                </Tr>
                            );
                        })}
                    </Tbody>
                </TableComposable>
            </Card>
        </Modal>
    );
}

export default AffectedComponentsModal;
