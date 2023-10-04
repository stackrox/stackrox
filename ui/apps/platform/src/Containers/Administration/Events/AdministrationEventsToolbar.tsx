import React, { ReactElement } from 'react';
import {
    Pagination,
    Text,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';
import pluralize from 'pluralize';

export type AdministrationEventsToolbarProps = {
    count: number;
    page: number;
    perPage: number;
    setPage: (newPage: number) => void;
    setPerPage: (newPerPage: number) => void;
};

function AdministrationEventsToolbar({
    count,
    page,
    perPage,
    setPage,
    setPerPage,
}: AdministrationEventsToolbarProps): ReactElement {
    const countText = `${count} ${pluralize('event', count)} found`;

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup>
                    <ToolbarItem variant="label">
                        <Text>{countText}</Text>
                    </ToolbarItem>
                </ToolbarGroup>
                <ToolbarGroup alignment={{ default: 'alignRight' }}>
                    <ToolbarItem variant="pagination">
                        <Pagination
                            itemCount={count}
                            page={page}
                            perPage={perPage}
                            onSetPage={(_, newPage) => setPage(newPage)}
                            onPerPageSelect={(_, newPerPage) => {
                                setPage(1);
                                setPerPage(newPerPage);
                            }}
                        />
                    </ToolbarItem>
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdministrationEventsToolbar;
