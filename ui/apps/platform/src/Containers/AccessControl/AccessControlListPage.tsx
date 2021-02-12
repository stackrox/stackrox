import React, { ReactElement, ReactNode } from 'react';

import Table from 'Components/Table';
import { AccessControlEntityType } from 'constants/entityTypes';

import { Column, AccessControlRow } from './accessControlTypes';
import AccessControlListHeader from './AccessControlListHeader';
import AccessControlPageHeader from './AccessControlPageHeader';

export type AccessControlListPageProps = {
    children: ReactNode;
    columns: Column[];
    entityType: AccessControlEntityType;
    rows: AccessControlRow[];
    selectedRowId?: string;
    setSelectedRowId: (id: string) => void;
};

function AccessControlListPage({
    // children,
    columns,
    entityType,
    rows,
    selectedRowId,
    setSelectedRowId,
}: AccessControlListPageProps): ReactElement {
    // TODO table is not yet interactive
    const disableSortRemove = true;
    const noDataText = '';
    const serverSidePagination = false;
    const sort = [];
    function onClickNew() {}
    function onRowClickHandler({ id }) {
        setSelectedRowId(id);
    }
    function onSortedChange() {}

    // TODO render entity in side panel
    // TODO factor out divs as presentation components?
    return (
        <>
            <AccessControlPageHeader currentType={entityType} />
            <div className="flex flex-1 h-full relative z-0">
                <div
                    className="flex flex-col border-r border-base-400 w-full h-full"
                    data-testid="main-panel"
                >
                    <AccessControlListHeader
                        entityCount={rows.length}
                        entityType={entityType}
                        isEntity={!!selectedRowId}
                        onClickNew={onClickNew}
                    />
                    <div className="h-full overflow-y-auto">
                        <Table
                            rows={rows}
                            columns={columns}
                            onRowClick={onRowClickHandler}
                            id="capture-list"
                            selectedRowId={selectedRowId}
                            noDataText={noDataText}
                            manual={serverSidePagination}
                            sorted={sort}
                            onSortedChange={onSortedChange}
                            disableSortRemove={disableSortRemove}
                        />
                    </div>
                </div>
                {/* side panel for selected entity */}
            </div>
        </>
    );
}

export default AccessControlListPage;
