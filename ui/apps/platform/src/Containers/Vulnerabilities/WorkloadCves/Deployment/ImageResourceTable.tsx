import React from 'react';
import { TableComposable, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';

import { UseURLSortResult } from 'hooks/useURLSort';
import { graphql } from 'generated/graphql-codegen';
import { ImageResourcesFragment } from 'generated/graphql-codegen/graphql';
import DateDistanceTd from '../components/DatePhraseTd';
import EmptyTableResults from '../components/EmptyTableResults';
import ImageNameTd from '../components/ImageNameTd';

export const imageResourcesFragment = graphql(/* GraphQL */ `
    fragment ImageResources on Deployment {
        images(query: $query, pagination: $pagination) {
            id
            name {
                registry
                remote
                tag
            }
            deploymentCount(query: $query)
            operatingSystem
            scanTime
        }
    }
`);

export type ImageResourceTableProps = {
    deployment: ImageResourcesFragment;
    getSortParams: UseURLSortResult['getSortParams'];
};

function ImageResourceTable({ deployment, getSortParams }: ImageResourceTableProps) {
    return (
        <TableComposable borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Image')}>Name</Th>
                    <Th>Image status</Th>
                    <Th>Image OS</Th>
                    <Th>Created</Th>
                </Tr>
            </Thead>
            {deployment.images.length === 0 && <EmptyTableResults colSpan={4} />}
            {deployment.images.map(({ id, name, deploymentCount, operatingSystem, scanTime }) => {
                return (
                    <Tbody
                        key={id}
                        style={{
                            borderBottom: '1px solid var(--pf-c-table--BorderColor)',
                        }}
                    >
                        <Tr>
                            <Td>{name ? <ImageNameTd id={id} name={name} /> : 'NAME UNKNOWN'}</Td>
                            {/* Given that this is in the context of a deployment, when would `deploymentCount` ever be less than zero? */}
                            <Td>{deploymentCount > 0 ? 'Active' : 'Inactive'}</Td>
                            <Td>{operatingSystem}</Td>
                            <Td>
                                <DateDistanceTd date={scanTime} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </TableComposable>
    );
}

export default ImageResourceTable;
