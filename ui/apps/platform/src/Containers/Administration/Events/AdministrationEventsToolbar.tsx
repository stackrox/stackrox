import React, { ReactElement } from 'react';
import {
    Pagination,
    Toolbar,
    ToolbarContent,
    ToolbarGroup,
    ToolbarItem,
} from '@patternfly/react-core';

export type AdministrationEventsToolbarProps = {
    count: string; // int64
};

function AdministrationEventsToolbar({ count }: AdministrationEventsToolbarProps): ReactElement {
    // TODO table filters and pagination
    const page = 1;
    const perPage = 10;

    return (
        <Toolbar>
            <ToolbarContent>
                <ToolbarGroup alignment={{ default: 'alignRight' }}>
                    <ToolbarItem variant="pagination">
                        <Pagination itemCount={Number(count) ?? 0} page={page} perPage={perPage} />
                    </ToolbarItem>
                </ToolbarGroup>
            </ToolbarContent>
        </Toolbar>
    );
}

export default AdministrationEventsToolbar;
