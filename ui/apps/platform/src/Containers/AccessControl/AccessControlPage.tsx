import React, { ReactElement, ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { Plus } from 'react-feather';
import pluralize from 'pluralize';

import PageHeader from 'Components/PageHeader';
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

import { getEntityPath } from './accessControlPaths';
import { AccessControlEntity, Column } from './accessControlTypes';

const linkTypes: AccessControlEntityType[] = [
    'AUTH_PROVIDER',
    'ROLE',
    'PERMISSION_SET',
    'ACCESS_SCOPE',
];

export type AccessControlPageProps = {
    children: ReactNode;
    columns: Column[];
    entityType: AccessControlEntityType;
    onClickNew: () => void;
    rows: AccessControlEntity[];
    selectedRowId?: string;
    setSelectedRowId: (id: string) => void;
};

function AccessControlPage({
    children,
    columns,
    entityType,
    onClickNew,
    rows,
    selectedRowId,
    setSelectedRowId,
}: AccessControlPageProps): ReactElement {
    const entityLabel = accessControlLabels[entityType];
    const textHead = `${rows.length} ${pluralize(entityLabel, rows.length)}`;
    const textNew = `New ${entityLabel}`;

    // TODO search filter is not yet interactive
    const searchOptions = [];
    const availableCategories = [];
    const autoFocusSearchInput = true;

    const disableSortRemove = true;
    const noDataText = '';
    const serverSidePagination = false;
    const sort = [];
    function onRowClickHandler({ id }) {
        setSelectedRowId(id);
    }
    function onSortedChange() {}

    // TODO are links disabled when editing an entity?
    return (
        <>
            <PageHeader
                header="Access Control"
                subHeader="Configure authentication, permissions, and scope"
                classes="pr-0"
            >
                <div className="flex flex-1 items-center justify-end">
                    {linkTypes.map((linkType) => {
                        const contrastClassNames =
                            entityType === linkType
                                ? 'bg-base-200 border-base-600 font-700'
                                : 'hover:bg-base-200 hover:border-base-600 border-base-400 font-600';
                        return (
                            <Link
                                key={entityType}
                                to={getEntityPath(linkType)}
                                className={`border-2 ${contrastClassNames} leading-none px-2 py-1 mr-4 rounded-full text-base-600 uppercase`}
                                data-testid={linkType}
                            >
                                {pluralize(accessControlLabels[linkType])}
                            </Link>
                        );
                    })}
                </div>
            </PageHeader>
            <PageBody>
                <PanelNew testid="main-panel">
                    <PanelHead>
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
                                className="btn btn-base h-10 ml-2 mr-4 whitespace-nowrap"
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
                {children}
            </PageBody>
        </>
    );
}

export default AccessControlPage;
