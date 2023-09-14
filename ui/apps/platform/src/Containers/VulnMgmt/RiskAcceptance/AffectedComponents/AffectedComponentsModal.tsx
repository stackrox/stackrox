import React, { ReactElement, useContext, useState } from 'react';
import {
    Card,
    CodeBlock,
    Grid,
    GridItem,
    InputGroup,
    Modal,
    ModalVariant,
    TextInput,
} from '@patternfly/react-core';
import {
    ExpandableRowContent,
    TableComposable,
    Tbody,
    Td,
    Th,
    Thead,
    Tr,
} from '@patternfly/react-table';

import entityTypes from 'constants/entityTypes';
import workflowStateContext from 'Containers/workflowStateContext';
import { EmbeddedImageScanComponent } from '../imageVulnerabilities.graphql';

export type AffectedComponentsModalProps = {
    cveName: string;
    isOpen: boolean;
    components: EmbeddedImageScanComponent[];
    onClose: () => void;
};

function AffectedComponentsModal({
    cveName = 'Unknown CVE',
    isOpen,
    components,
    onClose,
}: AffectedComponentsModalProps): ReactElement {
    const workflowState = useContext(workflowStateContext);
    const [inputValue, setInputValue] = useState('');
    const [expandedComponentIds, setExpandedComponentIds] = React.useState<string[]>([]);
    const setComponentIdExpanded = (component, isExpanding = true) =>
        setExpandedComponentIds((prevExpanded) => {
            const otherExpandedComponentIds = prevExpanded.filter((id) => id !== component.id);
            return isExpanding
                ? ([...otherExpandedComponentIds, component.id] as string[])
                : otherExpandedComponentIds;
        });
    const isComponentExpanded = (component) => expandedComponentIds.includes(component.id);

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
            title={`Components affected by ${cveName}`}
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
                            <Td />
                            <Th>Component</Th>
                            <Th>Version</Th>
                            <Th>Fixed in</Th>
                        </Tr>
                    </Thead>
                    {filteredComponents.map((component, rowIndex) => {
                        const componentURL = workflowState
                            .pushList(entityTypes.IMAGE_COMPONENT)
                            .pushListItem(component.id)
                            .toUrl();
                        return (
                            <Tbody key={component.id} isExpanded={isComponentExpanded(component)}>
                                <Tr key={component.name}>
                                    <Td
                                        expand={{
                                            rowIndex,
                                            isExpanded: isComponentExpanded(component),
                                            onToggle: () =>
                                                setComponentIdExpanded(
                                                    component,
                                                    !isComponentExpanded(component)
                                                ),
                                            expandId: 'affected-components-expandable-toggle',
                                        }}
                                    />
                                    <Td dataLabel="Component">
                                        <a href={componentURL} target="_blank" rel="noreferrer">
                                            {component.name}
                                        </a>
                                    </Td>
                                    <Td dataLabel="Version">{component.version}</Td>
                                    <Td dataLabel="Fixed in">{component.fixedIn || '-'}</Td>
                                </Tr>
                                <Tr isExpanded={isComponentExpanded(component)}>
                                    <Td
                                        dataLabel="Dockerfile line where component is added"
                                        colSpan={4}
                                    >
                                        <ExpandableRowContent>
                                            <CodeBlock>
                                                <Grid hasGutter>
                                                    <GridItem span={1}>
                                                        {component?.dockerfileLine?.line}
                                                    </GridItem>
                                                    <GridItem span={2}>
                                                        {component?.dockerfileLine?.instruction}
                                                    </GridItem>
                                                    <GridItem span={9}>
                                                        {component?.dockerfileLine?.value}
                                                    </GridItem>
                                                </Grid>
                                            </CodeBlock>
                                        </ExpandableRowContent>
                                    </Td>
                                </Tr>
                            </Tbody>
                        );
                    })}
                </TableComposable>
            </Card>
        </Modal>
    );
}

export default AffectedComponentsModal;
