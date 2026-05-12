import { Table, Tbody, Td, Th, Thead, Tr } from '@patternfly/react-table';
import { gql } from '@apollo/client';

import type { UseURLSortResult } from 'hooks/useURLSort';
import DateDistance from 'Components/DateDistance';
import EmptyTableResults from '../components/EmptyTableResults';
import ImageNameLink from '../components/ImageNameLink';

export type ImageResources = {
    imageCount: number;
    images: {
        id: string;
        digest: string;
        name: {
            registry: string;
            remote: string;
            tag: string;
        } | null;
        deploymentCount: number;
        activeDeploymentCount: number;
        operatingSystem: string;
        scanTime: string | null;
    }[];
};

export const imageResourcesFragment = gql`
    fragment ImageResources on Deployment {
        imageCount(query: $query)
        images(query: $query, pagination: $pagination) {
            id
            digest
            name {
                registry
                remote
                tag
            }
            deploymentCount(query: $query)
            activeDeploymentCount: deploymentCount(query: $activeDeploymentQuery)
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
        <Table variant="compact">
            <Thead noWrap>
                <Tr>
                    <Th sort={getSortParams('Image')}>Name</Th>
                    <Th>Image status</Th>
                    <Th>Image OS</Th>
                    <Th>Created</Th>
                </Tr>
            </Thead>
            {data.images.length === 0 && <EmptyTableResults colSpan={4} />}
            {data.images.map(({ id, digest, name, activeDeploymentCount, operatingSystem, scanTime }) => {
                return (
                    <Tbody key={id}>
                        <Tr>
                            <Td dataLabel="Name" width={50}>
                                {name ? (
                                    <ImageNameLink id={id} name={name} digest={digest} />
                                ) : (
                                    'NAME UNKNOWN'
                                )}
                            </Td>
                            <Td dataLabel="Image status">
                                {activeDeploymentCount > 0 ? 'Active' : 'Inactive'}
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
