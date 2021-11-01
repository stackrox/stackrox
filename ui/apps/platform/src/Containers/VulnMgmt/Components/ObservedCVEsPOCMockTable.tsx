/* eslint-disable no-nested-ternary */
/* eslint-disable react/no-array-index-key */
import React from 'react';
import { TableComposable, Thead, Tbody, Tr, Th, Td } from '@patternfly/react-table';

const defaultActions = [
    {
        title: 'Defer CVE',
        onClick: (event) => {
            event.preventDefault();
        },
    },
    {
        title: 'Mark as False Positive',
        onClick: (event) => {
            event.preventDefault();
        },
    },
    {
        isSeparator: true,
    },
    {
        title: 'Reject deferral',
        onClick: (event) => {
            event.preventDefault();
        },
    },
];
const columns = ['CVE', 'Fixable', 'Severity', 'CVSS Score', 'Affected Components', 'Discovered'];
const rows = [
    ['CVE-2014-232', 'No', 'Medium', '5.8', '2 components', '3 days ago'],
    ['CVE-2019-5953', 'Yes', 'Critical', '9.8', '1 component', '2 days ago'],
    ['CVE-2017-13090', 'Yes', 'Important', '8.8', '1 component', '1 day ago'],
    ['CVE-2016-7098', 'Yes', 'Important', '8.1', '1 component', '8 days ago'],
    ['CVE-2018-0494', 'Yes', 'Medium', '6.5', '3 components', '12 days ago'],
];

function ObservedCVEsPOCMockTable() {
    return (
        <TableComposable aria-label="Actions table">
            <Thead>
                <Tr>
                    <Th>{columns[0]}</Th>
                    <Th>{columns[1]}</Th>
                    <Th>{columns[2]}</Th>
                    <Th>{columns[3]}</Th>
                    <Th>{columns[4]}</Th>
                    <Th />
                </Tr>
            </Thead>
            <Tbody>
                {rows.map((row, rowIndex) => (
                    <Tr key={rowIndex}>
                        {row.map((cell, cellIndex) => (
                            <Td key={`${rowIndex}_${cellIndex}`} dataLabel={columns[cellIndex]}>
                                {cell}
                            </Td>
                        ))}
                        <Td
                            key={`${rowIndex}_5`}
                            actions={{
                                items: defaultActions,
                            }}
                        />
                    </Tr>
                ))}
            </Tbody>
        </TableComposable>
    );
}

export default ObservedCVEsPOCMockTable;
