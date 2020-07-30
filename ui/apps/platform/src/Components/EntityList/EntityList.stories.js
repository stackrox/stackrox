/* eslint-disable no-use-before-define */
import React from 'react';
import { MemoryRouter } from 'react-router-dom';
import { createMemoryHistory } from 'history';
import { Provider } from 'react-redux';
import pluralize from 'pluralize';

import LabelChip from 'Components/LabelChip';
import { defaultHeaderClassName, defaultColumnClassName } from 'Components/Table';
import TableCellLink from 'Components/TableCellLink';
import entityTypes from 'constants/entityTypes';
import configureStore from 'store/configureStore';

import EntityList from './EntityList';

const tableColumns = getTableColumns();
const rowData = getRowData();

export default {
    title: 'EntityList',
    component: EntityList,
};

const history = createMemoryHistory('/');
const store = configureStore(undefined, history);

export const basicEntityList = () => (
    <Provider store={store}>
        <MemoryRouter>
            <EntityList
                entityType={entityTypes.DEPLOYMENT}
                idAttribute="id"
                rowData={rowData}
                tableColumns={tableColumns}
            />
        </MemoryRouter>
    </Provider>
);

function getTableColumns() {
    return [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: `Deployment`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name',
        },
        {
            Header: `Cluster`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'clusterName',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { clusterName } = original;
                const url = 'https://stackrow.com';
                return <TableCellLink pdf={pdf} url={url} text={clusterName} />;
            },
        },
        {
            Header: `Namespace`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'namespace',
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { namespace } = original;
                const url = 'https://wikipedia.org';
                return <TableCellLink pdf={pdf} url={url} text={namespace} />;
            },
        },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original }) => {
                const { policyStatus } = original;
                return policyStatus === 'pass' ? 'Pass' : <LabelChip text="Fail" type="alert" />;
            },
            id: 'policyStatus',
            accessor: 'policyStatus',
        },
        {
            Header: `Images`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            // eslint-disable-next-line
            Cell: ({ original, pdf }) => {
                const { imageCount } = original;
                if (imageCount === 0) return 'No images';
                const url = 'https://google.com';
                return (
                    <TableCellLink
                        pdf={pdf}
                        url={url}
                        text={`${imageCount} ${pluralize('image', imageCount)}`}
                    />
                );
            },
            accessor: 'imageCount',
        },
    ];
}

function getRowData() {
    return [
        {
            id: 'ad49e750-e9bb-11e9-93e8-42010a8a0114',
            name: 'prometheus-to-sd',
            clusterName: 'remote',
            clusterId: '14304fbf-2e15-4752-95e0-dc60b79f1dad',
            namespace: 'kube-system',
            namespaceId: 'a019d8de-e9bb-11e9-93e8-42010a8a0114',
            imageCount: 1,
            policyStatus: 'pass',
        },
        {
            id: 'b2ed2df0-e9bb-11e9-93e8-42010a8a0114',
            name: 'nvidia-gpu-device-plugin',
            clusterName: 'remote',
            clusterId: '14304fbf-2e15-4752-95e0-dc60b79f1dad',
            namespace: 'kube-system',
            namespaceId: 'a019d8de-e9bb-11e9-93e8-42010a8a0114',
            imageCount: 1,
            policyStatus: 'fail',
        },
        {
            id: 'b2fc068c-e9bb-11e9-93e8-42010a8a0114',
            name: 'ip-masq-agent',
            clusterName: 'remote',
            clusterId: '14304fbf-2e15-4752-95e0-dc60b79f1dad',
            namespace: 'kube-system',
            namespaceId: 'a019d8de-e9bb-11e9-93e8-42010a8a0114',
            imageCount: 1,
            policyStatus: 'fail',
        },
    ];
}
