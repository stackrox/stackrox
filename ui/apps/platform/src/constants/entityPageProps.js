import PropTypes from 'prop-types';

export const entityPagePropTypes = {
    entityId: PropTypes.string.isRequired,
    listEntityType1: PropTypes.string,
    entityType1: PropTypes.string,
    entityId1: PropTypes.string,
    entityType2: PropTypes.string,
    entityListType2: PropTypes.string,
    entityId2: PropTypes.string,
    query: PropTypes.shape({}),
    sidePanelMode: PropTypes.bool,
    controlResult: PropTypes.shape({}),
    original: PropTypes.shape({}),
};

export const entityPageDefaultProps = {
    listEntityType1: null,
    entityType1: null,
    entityId1: null,
    entityType2: null,
    entityListType2: null,
    entityId2: null,
    query: null,
    sidePanelMode: false,
    controlResult: null,
    original: null,
};

export const entityListPropTypes = {
    className: PropTypes.string,
    selectedRowId: PropTypes.string,
    query: PropTypes.shape({}),
    data: PropTypes.arrayOf(PropTypes.shape({})),
};

export const entityListDefaultprops = {
    className: '',
    selectedRowId: null,
    query: null,
    data: null,
};

// TODO: standardize on entityId and search props from legacy id and query.
export const entityComponentPropTypes = {
    id: PropTypes.string,
    entityId: PropTypes.string,
    query: PropTypes.shape({}),
    search: PropTypes.shape({}),
    entityListType: PropTypes.string,
    entityContext: PropTypes.shape({}),
};

export const entityComponentDefaultProps = {
    entityListType: null,
    contextEntityType: null,
    contextEntityId: null,
    entityContext: {},
    query: null,
    search: null,
    id: null,
    entityid: null,
};

export const workflowListPropTypes = {
    data: PropTypes.arrayOf(PropTypes.shape({})),
    totalResults: PropTypes.number,
    selectedRowId: PropTypes.string,
    search: PropTypes.shape({}),
    sort: PropTypes.arrayOf(PropTypes.shape({})),
    page: PropTypes.number,
};

export const workflowListDefaultProps = {
    data: null,
    totalResults: null,
    search: null,
    entityContext: {},
    sort: null,
    page: 0,
    selectedRowId: null,
};

export const workflowEntityPropTypes = {
    entityId: PropTypes.string.isRequired,
    entityListType: PropTypes.string,
    entityContext: PropTypes.shape({}),
    search: PropTypes.shape({}),
    sort: PropTypes.arrayOf(PropTypes.shape({})),
    page: PropTypes.number,
};

export const workflowEntityDefaultProps = {
    entityListType: null,
    entityContext: {},
    search: null,
    sort: [],
    page: 1,
};
