import React, { useContext } from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';
import camelCase from 'lodash/camelCase';

import TableCellLink from 'Components/TableCellLink';
import workflowStateContext from 'Containers/workflowStateContext';
import entityLabels from 'messages/entity';

const TableCountLink = ({ selectedRowId, entityType, textOnly, count, entityTypeText, search }) => {
    const workflowState = useContext(workflowStateContext);

    const type = entityTypeText || entityLabels[entityType];
    if (count === 0) return `No ${pluralize(type)}`;

    const text = `${count} ${pluralize(type, count)}`;
    if (textOnly) return <span data-testid={`${type}CountText`}>{text}</span>;

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
};

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
