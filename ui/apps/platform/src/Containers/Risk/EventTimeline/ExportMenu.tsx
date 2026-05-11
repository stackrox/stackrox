import { List } from 'react-feather';

import downloadCSV from 'services/CSVDownloadService';
import Menu from 'Components/Menu';
import type { MenuOption } from 'Components/Menu';

type ExportMenuProps = {
    fileName: string;
    csvEndpoint?: string;
    csvQueryString?: string;
};

const ExportMenu = ({ fileName, csvEndpoint, csvQueryString = '' }: ExportMenuProps) => {
    const options: MenuOption[] = [];
    if (csvEndpoint) {
        options.push({
            className: '',
            icon: <List className="h-4 w-4 text-base-600" />,
            label: 'Download CSV',
            onClick: () => {
                return downloadCSV(fileName, csvEndpoint, csvQueryString);
            },
        });
    }
    return (
        <Menu
            className="h-full min-w-30"
            menuClassName="bg-base-100 min-w-28"
            buttonClass="btn btn-base"
            buttonText="Export"
            options={options}
            disabled={false}
            dataTestId="export-menu"
        />
    );
};

export default ExportMenu;
