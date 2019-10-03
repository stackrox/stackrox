import React from 'react';
import { MemoryRouter } from 'react-router-dom';

import TableCellLink from './TableCellLink';

export default {
    title: 'TableCellLink',
    component: TableCellLink
};

export const basicTableCellLink = () => (
    <MemoryRouter>
        <TableCellLink
            pdf={false}
            url="/main/configmanagement/cluster/88d17fde-3b80-48dc-a4f3-1c8068e95f28"
            text="remote"
        />
    </MemoryRouter>
);

export const withPDFflagSet = () => (
    <MemoryRouter>
        <TableCellLink
            pdf
            url="/main/configmanagement/cluster/88d17fde-3b80-48dc-a4f3-1c8068e95f28"
            text="cluster_on_pdf"
        />
    </MemoryRouter>
);

export const inATable = () => (
    <MemoryRouter>
        <div className="ReactTable flex flex-1 overflow-auto border-0 w-full h-full text-base">
            <div className="rt-table" role="grid">
                <div className="rt-thead -header" style={{ minidth: '800px' }}>
                    <div className="rt-tr" role="row">
                        <div
                            className="rt-th hidden -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>Id</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>Cluster</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>K8S Version</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>Policy Status</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>CIS Controls</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>Users &amp; Groups</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>Service Accounts</div>
                        </div>
                        <div
                            className="rt-th w-1/8 px-2 py-4 pb-3 font-700 text-base-600 hover:bg-primary-200 hover:z-1 hover:text-primary-700 select-none relative text-left border-r-0 leading-normal -cursor-pointer"
                            role="columnheader"
                            tabIndex="-1"
                            style={{ flex: '100 0 auto', width: '100px' }}
                        >
                            <div>Roles</div>
                        </div>
                    </div>
                </div>
                <div className="rt-tbody" style={{ minidth: '800px' }}>
                    <div className="rt-tr-group" role="rowgroup">
                        <div className="rt-tr    -odd" role="row">
                            <div
                                className="rt-td hidden"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                88d17fde-3b80-48dc-a4f3-1c8068e95f28
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                remote
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                v1.14.6
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                <span className="border px-2 rounded bg-alert-200 border-alert-400 text-alert-800">
                                    Fail
                                </span>
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                <span className="border px-2 rounded bg-alert-200 border-alert-400 text-alert-800">
                                    No Controls
                                </span>
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                <TableCellLink
                                    url="/main/configmanagement/clusters/88d17fde-3b80-48dc-a4f3-1c8068e95f28/subjects"
                                    text="10 Users & Groups"
                                />
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                <TableCellLink
                                    url="/main/configmanagement/clusters/88d17fde-3b80-48dc-a4f3-1c8068e95f28/serviceaccounts"
                                    text="40 Service Accounts"
                                />
                            </div>
                            <div
                                className="rt-td w-1/8 p-2 flex items-center font-600 text-base-600 text-left border-r-0 leading-normal"
                                role="gridcell"
                                style={{ flex: '100 0 auto', width: '100px' }}
                            >
                                <TableCellLink
                                    url="/main/configmanagement/clusters/88d17fde-3b80-48dc-a4f3-1c8068e95f28/roles"
                                    text="80 Roles"
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </MemoryRouter>
);
