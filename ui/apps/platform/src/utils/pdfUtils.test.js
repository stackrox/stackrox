import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import ReactDOMServer from 'react-dom/server';
import createPDFTable from './pdfUtils';

vi.mock('react-dom/server');

describe('createPDFTable', () => {
    let parentElement;

    beforeEach(() => {
        parentElement = document.createElement('div');
        parentElement.id = 'test-pdf-parent';
        document.body.appendChild(parentElement);

        ReactDOMServer.renderToString.mockReturnValue('<span>rendered-value</span>');
    });

    afterEach(() => {
        document.body.removeChild(parentElement);
        vi.clearAllMocks();
    });

    describe('maintain correct column indices in flat and cluster-grouped layouts', () => {
        it('should render flat controls without grouping', () => {
            const tableData = [
                { control: '4.1.1', description: 'Image Vulnerabilities', compliance: '100%' },
                { control: '4.1.2', description: 'Image configuration defects', compliance: '0%' },
            ];

            const complianceRenderer = vi.fn();
            const tableColumns = [
                { accessor: 'control', Header: 'Control', className: '' },
                { accessor: 'description', Header: 'Description', className: '' },
                {
                    accessor: 'compliance',
                    Header: 'Compliance',
                    className: '',
                    Cell: complianceRenderer,
                },
            ];

            createPDFTable(tableData, 'control', {}, 'test-pdf-parent', tableColumns);

            // Verify columns are in correct order
            const table = document.getElementById('pdf-table');
            const headerRow = table.querySelector('tbody tr');
            const headers = Array.from(headerRow.querySelectorAll('th')).map((h) => h.textContent);
            expect(headers).toEqual(['Control', 'Description', 'Compliance']);

            // Verify compliance renderer was called
            expect(complianceRenderer).toHaveBeenCalled();
        });

        it('should render cluster-grouped controls', () => {
            const tableData = [
                {
                    name: 'misha-fips-test-cluster',
                    groupId: 1,
                    rows: [
                        {
                            control: '4.1.1',
                            description: 'Image Vulnerabilities',
                            compliance: '100%',
                        },
                        {
                            control: '4.1.2',
                            description: 'Image configuration defects',
                            compliance: '0%',
                        },
                    ],
                },
            ];

            const complianceRenderer = vi.fn();
            const tableColumns = [
                { accessor: 'control', Header: 'Control', className: '' },
                { accessor: 'description', Header: 'Description', className: '' },
                {
                    accessor: 'compliance',
                    Header: 'Compliance',
                    className: '',
                    Cell: complianceRenderer,
                },
            ];

            createPDFTable(
                tableData,
                'control',
                { groupBy: 'cluster' },
                'test-pdf-parent',
                tableColumns
            );

            // Verify cluster column is prepended first
            const table = document.getElementById('pdf-table');
            const headerRow = table.querySelector('tbody tr');
            const headers = Array.from(headerRow.querySelectorAll('th')).map((h) => h.textContent);
            expect(headers).toEqual(['Cluster', 'Control', 'Description', 'Compliance']);

            // Verify compliance renderer still works with correct column offset
            expect(complianceRenderer).toHaveBeenCalled();
        });
    });
});
