import React from 'react';
import PropTypes from 'prop-types';
import * as Icon from 'react-feather';

import List from 'Components/List';
import { PanelNew, PanelBody, PanelHead, PanelTitle } from 'Components/Panel';
import { defaultColumnClassName, rtTrActionsClassName } from 'Components/Table';
import RowActionButton from 'Components/RowActionButton';

const SideBar = ({
    onSelectRow,
    onCancel,
    onDelete,
    type,
    header,
    rows,
    selected,
    addRowButton,
}) => {
    function onRowSelectHandler() {
        return (row) => {
            onSelectRow(row);
            if (onCancel) {
                onCancel();
            }
        };
    }

    function onDeleteHandler(row) {
        return (e) => {
            e.stopPropagation();
            onDelete(row);
        };
    }

    function renderRowActionButtons(row) {
        if (!onDelete || row.noAction) {
            return null;
        }
        return (
            <div className="border-2 border-base-400 bg-base-100 flex">
                <RowActionButton
                    text={`Delete ${type}`}
                    icon={<Icon.Trash2 className="my-1 h-4 w-4" />}
                    onClick={onDeleteHandler(row)}
                />
            </div>
        );
    }

    const columns = [
        {
            id: 'name',
            accessor: 'name',
            className: `${defaultColumnClassName}`,
        },
        {
            accessor: '',
            headerClassName: 'hidden',
            className: rtTrActionsClassName,
            Cell: ({ original }) => renderRowActionButtons(original),
        },
    ];

    return (
        <PanelNew testid="panel">
            <PanelHead>
                <PanelTitle isUpperCase testid="panel-header" text={header} />
            </PanelHead>
            <PanelBody>
                <div className="overflow-auto table-reset-padding">
                    <List
                        columns={columns}
                        rows={rows}
                        selectRow={onRowSelectHandler()}
                        selectedListItem={selected}
                        selectedIdAttribute="name"
                    />
                </div>
                {addRowButton && (
                    <div className="flex items-center justify-center p-4 border-t border-base-300">
                        {addRowButton}
                    </div>
                )}
            </PanelBody>
        </PanelNew>
    );
};

SideBar.propTypes = {
    header: PropTypes.string.isRequired,
    rows: PropTypes.arrayOf(PropTypes.shape({})).isRequired,
    selected: PropTypes.shape({}),
    onSelectRow: PropTypes.func.isRequired,
    addRowButton: PropTypes.node,
    onCancel: PropTypes.func,
    onDelete: PropTypes.func,
    type: PropTypes.string.isRequired,
};

SideBar.defaultProps = {
    onCancel: null,
    onDelete: null,
    selected: null,
    addRowButton: null,
};

export default SideBar;
