import React, { ReactElement, useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import camelCase from 'lodash/camelCase';

import TableCellLink from 'Components/TableCellLink';
import workflowStateContext from 'Containers/workflowStateContext';
import entityLabels from 'messages/entity';

type TableCountLinkProps = {
    selectedRowId: string;
    entityType: string;
    textOnly?: boolean;
    count: number;
    entityTypeText?: string;
    search?: Record<string, boolean>;
};

function TableCountLink({
    selectedRowId,
    entityType,
    textOnly = false,
    count,
    entityTypeText = '',
    search = {},
}: TableCountLinkProps): ReactElement {
    const workflowState = useContext(workflowStateContext);

    // TODO type cast required until inconsistency is resolved between keys in constants/entityTypes and messages/common:
    const type = entityTypeText || (entityLabels[entityType] as string);
    if (count === 0) {
        return <div>No {pluralize(type)}</div>;
    }

    const text = `${count} ${pluralize(type, count)}`;
    if (textOnly) {
        return <div data-testid={`${type}CountText`}>{text}</div>;
    }

    const newState = workflowState.pushListItem(selectedRowId).pushList(entityType);
    const urlWithSearch = newState.setSearch(search).toUrl();

    return (
        <TableCellLink
            pdf={textOnly}
            url={urlWithSearch}
            text={text}
            dataTestId={`${camelCase(type)}CountLink`}
        />
    );
}

TableCountLink.propTypes = {
    entityType: PropTypes.string.isRequired,
    selectedRowId: PropTypes.string.isRequired,
    textOnly: PropTypes.bool,
    count: PropTypes.number.isRequired,
    entityTypeText: PropTypes.string,
    search: PropTypes.shape({}),
};

TableCountLink.defaultProps = {
    textOnly: false,
    entityTypeText: null,
    search: {},
};

export default TableCountLink;
