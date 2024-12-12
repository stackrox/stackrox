import React from 'react';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import { UseURLSortResult } from 'hooks/useURLSort';
import DateDistance from 'Components/DateDistance';
import EmptyTableResults from '../components/EmptyTableResults';
import ImageNameLink from '../components/ImageNameLink';

export type ImageResources = {
    imageCount: number;
    images: {
        id: string;
        name: {
            registry: string;
            remote: string;
            tag: string;
        } | null;
        deploymentCount: number;
        operatingSystem: string;
        scanTime: string | null;
    }[];
};

export const imageResourcesFragment = gql`
    fragment ImageResources on Deployment {
        imageCount(query: $query)
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
`;

export type ImageResourceTableProps = {
    data: ImageResources;
    getSortParams: UseURLSortResult['getSortParams'];
};

function ImageResourceTable({ data, getSortParams }: ImageResourceTableProps) {
    return (
        <Table borders={false} variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Image')}>Name</Th>
                    <Th>Image status</Th>
                    <Th>Image OS</Th>
                    <Th>Created</Th>
                </Tr>
            </Thead>
            {data.images.length === 0 && <EmptyTableResults colSpan={4} />}
            {data.images.map(({ id, name, deploymentCount, operatingSystem, scanTime }) => {
                return (
                    <Tbody
                        key={id}
                        style={{
                            borderBottom: '1px solid var(--pf-v5-c-table--BorderColor)',
                        }}
                    >
                        <Tr>
                            <Td dataLabel="Name" width={50}>
                                {name ? <ImageNameLink id={id} name={name} /> : 'NAME UNKNOWN'}
                            </Td>
                            {/* Given that this is in the context of a deployment, when would `deploymentCount` ever be less than zero? */}
                            <Td dataLabel="Image status">
                                {deploymentCount > 0 ? 'Active' : 'Inactive'}
                            </Td>
                            <Td dataLabel="Image OS">{operatingSystem}</Td>
                            <Td dataLabel="Created">
                                <DateDistance date={scanTime} />
                            </Td>
                        </Tr>
                    </Tbody>
                );
            })}
        </Table>
    );
}

export default ImageResourceTable;
