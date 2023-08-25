import React from 'react';
import { defaultHeaderClassName, defaultColumnClassName, wrapClassName } from 'Components/Table';
import entityTypes, { resourceTypes } from 'constants/entityTypes';
import PolicyStatusIconText from 'Components/PatternFly/IconText/PolicyStatusIconText';
import { format } from 'date-fns';
import dateTimeFormat from 'constants/dateTimeFormat';

const nodesAcrossControlsColumns = [
    {
        Header: 'Id',
        headerClassName: 'hidden',
        className: 'hidden',
        accessor: 'id',
    },
    {
        Header: `Node`,
        headerClassName: `w-1/3 ${defaultHeaderClassName}`,
        className: `w-1/3 ${defaultColumnClassName}`,
        accessor: 'name',
    },
    {
        Header: `Cluster`,
        headerClassName: `w-1/3 ${defaultHeaderClassName}`,
        className: `w-1/3 ${defaultColumnClassName}`,
        accessor: 'clusterName',
    },
    {
        Header: `Control Status`,
        headerClassName: `w-1/8 ${defaultHeaderClassName}`,
        className: `w-1/8 ${defaultColumnClassName}`,
        Cell: ({ original, pdf }) => {
            return <PolicyStatusIconText isPass={original.passing} isTextOnly={pdf} />;
        },
    },
];

const imageColumns = [
    {
        expander: true,
        headerClassName: `w-1/8 ${defaultHeaderClassName} pointer-events-none`,
        className: 'w-1/8 pointer-events-none flex items-center justify-end',
        Expander: ({ isExpanded, ...rest }) => {
            if (!rest.original.components || rest.original.components.length === 0) {
                return '';
            }
            const className = 'rt-expander w-1 pt-2 pointer-events-auto';
            return <div className={`${className} ${isExpanded ? '-open' : ''}`} />;
        },
    },
    {
        accessor: 'instruction',
        Header: 'Instruction',
        headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
        className: `text-left pl-3 ${wrapClassName} ${defaultColumnClassName}`,
    },
    {
        accessor: 'value',
        Header: 'Value',
        headerClassName: `w-3/5 text-left ${wrapClassName} ${defaultHeaderClassName}`,
        className: `w-3/5 text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
    },
    {
        accessor: 'created',
        Header: 'Created',
        align: 'right',
        widthClassName: `text-left pr-3 ${wrapClassName} ${defaultHeaderClassName}`,
        className: `text-left pr-3 ${wrapClassName} ${defaultColumnClassName}`,
        Cell: ({ original }) => format(original.created, dateTimeFormat),
    },
    {
        accessor: 'components.length',
        Header: 'Components',
        headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
        className: `text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
    },
    {
        accessor: 'cvesCount',
        Header: 'CVEs',
        headerClassName: `text-left ${wrapClassName} ${defaultHeaderClassName}`,
        className: `text-left pl-3 word-break-all ${wrapClassName} ${defaultColumnClassName}`,
    },
];

const getDeploymentViolationsColumns = (entityContext) => {
    const columns = [
        {
            Header: 'Id',
            headerClassName: 'hidden',
            className: 'hidden',
            accessor: 'id',
        },
        {
            Header: `Deployment`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'name',
        },
        entityContext &&
        (entityContext[entityTypes.CLUSTER] || entityContext[entityTypes.NAMESPACE])
            ? null
            : {
                  Header: `Cluster`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'clusterName',
              },
        entityContext && entityContext[entityTypes.NAMESPACE]
            ? null
            : {
                  Header: `Namespace`,
                  headerClassName: `w-1/8 ${defaultHeaderClassName}`,
                  className: `w-1/8 ${defaultColumnClassName}`,
                  accessor: 'namespace',
              },
        {
            Header: `Policy Status`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            Cell: ({ pdf }) => <PolicyStatusIconText isPass={false} isTextOnly={pdf} />,
        },
        {
            Header: `Violation Time`,
            headerClassName: `w-1/8 ${defaultHeaderClassName}`,
            className: `w-1/8 ${defaultColumnClassName}`,
            accessor: 'violationTime',
            Cell: ({ original }) => format(original.time, dateTimeFormat),
        },
    ];
    return columns.filter((col) => col);
};

export const entityToColumns = {
    [resourceTypes.IMAGE]: imageColumns,
};

export const entityAcrossControlsColumns = {
    [resourceTypes.NODE]: nodesAcrossControlsColumns,
};

export const entityViolationsColumns = {
    [resourceTypes.DEPLOYMENT]: getDeploymentViolationsColumns,
};
