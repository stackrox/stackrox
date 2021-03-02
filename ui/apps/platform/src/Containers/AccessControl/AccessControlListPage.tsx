import React, { ReactElement, ReactNode } from 'react';
import { Plus } from 'react-feather';
import pluralize from 'pluralize';

import SidePanelAnimatedArea from 'Components/animations/SidePanelAnimatedArea';
import {
    PageBody,
    PanelNew,
    PanelBody,
    PanelHead,
    PanelHeadEnd,
    PanelTitle,
} from 'Components/Panel';
import PanelButton from 'Components/PanelButton';
import Table from 'Components/Table';
import URLSearchInput from 'Components/URLSearchInput';
import { AccessControlEntityType } from 'constants/entityTypes';
import { accessControlLabels } from 'messages/common';

import { AccessControlEntity, Column } from './accessControlTypes';
import AccessControlPageHeader from './AccessControlPageHeader';

export type AccessControlListPageProps = {
    children: ReactNode;
    columns: Column[];
    entityType: AccessControlEntityType;
    isDarkMode: boolean;
    rows: AccessControlEntity[];
    selectedRowId?: string;
    setSelectedRowId: (id: string) => void;
};

function AccessControlListPage({
    children,
    columns,
    entityType,
    isDarkMode,
    rows,
    selectedRowId,
    setSelectedRowId,
}: AccessControlListPageProps): ReactElement {
    const entityLabel = accessControlLabels[entityType];
    const textHead = `${rows.length} ${pluralize(entityLabel, rows.length)}`;
    const textNew = `New ${entityLabel}`;

    // TODO search filter is not yet interactive
    const searchOptions = [];
    const availableCategories = [];
    const autoFocusSearchInput = true;

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

    return (
        <>
            <AccessControlPageHeader currentType={entityType} />
            <PageBody>
                <PanelNew testid="main-panel">
                    <PanelHead isDarkMode={isDarkMode}>
                        <PanelTitle isUpperCase testid="head-text" text={textHead} />
                        <PanelHeadEnd>
                            <URLSearchInput
                                className="w-full"
                                categoryOptions={searchOptions}
                                categories={availableCategories}
                                autoFocus={autoFocusSearchInput}
                            />
                            <PanelButton
                                icon={<Plus className="h-4 w-4 ml-1" />}
                                tooltip={textNew}
                                className="btn btn-base ml-2 mr-4 whitespace-nowrap"
                                onClick={onClickNew}
                                disabled={!!selectedRowId}
                            >
                                {textNew}
                            </PanelButton>
                        </PanelHeadEnd>
                    </PanelHead>
                    <PanelBody>
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
                    </PanelBody>
                </PanelNew>
                <SidePanelAnimatedArea isDarkMode={isDarkMode} isOpen={!!selectedRowId}>
                    {children}
                </SidePanelAnimatedArea>
            </PageBody>
        </>
    );
}

export default AccessControlListPage;
