import React, { ReactElement } from 'react';
import { Link } from 'react-router-dom';
import { Flex, FlexItem, Modal, ModalVariant, Title } from '@patternfly/react-core';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import pluralize from 'pluralize';

import entityTypes from 'constants/entityTypes';
import { resourceLabels } from 'messages/common';
import { vulnManagementPath } from 'routePaths';
import { VulnerabilityRequest } from '../vulnerabilityRequests.graphql';

function EntityName({ entity, entityType }) {
    return entityType === entityTypes.IMAGE ? (
        <Link to={`${vulnManagementPath}/image/${entity.id as string}`}>
            {entity?.name?.fullName}
        </Link>
    ) : (
        <Flex direction={{ default: 'column' }}>
            <FlexItem className="pf-u-mb-0">
                <Link to={`${vulnManagementPath}/deployment/${entity.id as string}`}>
                    {entity.name}
                </Link>
            </FlexItem>
            <FlexItem className="pf-u-color-400 pf-u-font-size-xs">
                {`in "${entity.clusterName as string}/${entity.namespace as string}"`}
            </FlexItem>
        </Flex>
    );
}
export type ImpactedEntitiesModalProps = {
    isOpen: boolean;
    entityType: string;
    entities: VulnerabilityRequest['deployments'] | VulnerabilityRequest['images'];
    onClose: () => void;
};

function ImpactedEntitiesModal({
    isOpen,
    entityType,
    entities,
    onClose,
}: ImpactedEntitiesModalProps): ReactElement {
    const entityTypeName = pluralize(resourceLabels[entityType] ?? '', entities?.length);
    const header = (
        <Title headingLevel="h2">
            <span style={{ textTransform: 'capitalize' }}>{entityTypeName}</span> impacted by this
            request
        </Title>
    );

    return (
        <Modal variant={ModalVariant.small} header={header} isOpen={isOpen} onClose={onClose}>
            <TableComposable aria-label="Simple table" variant="compact">
                <Thead>
                    <Tr>
                        <Th modifier="fitContent">Name</Th>
                    </Tr>
                </Thead>
                <Tbody>
                    {entities.map((entity) => (
                        <Tr key={entity?.name ?? entity?.name?.fullName}>
                            <Td dataLabel="Name">
                                <EntityName entity={entity} entityType={entityType} />
                            </Td>
                        </Tr>
                    ))}
                </Tbody>
            </TableComposable>
        </Modal>
    );
}

export default ImpactedEntitiesModal;
