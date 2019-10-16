import React from 'react';

import ExportButton from 'Components/ExportButton';
import PoliciesTile from './PoliciesTile';
import CISControlsTile from './CISControlsTile';
import AppMenu from './AppMenu';
import RBACMenu from './RBACMenu';

const Header = () => {
    return (
        <div className="flex flex-1 justify-end">
            <PoliciesTile />
            <CISControlsTile />
            <div className="flex w-32">
                <AppMenu />
            </div>
            <div className="flex w-32">
                <RBACMenu />
            </div>
            <div className="flex items-center self-center">
                <ExportButton
                    fileName="Config Mangement Dashboard Report"
                    type={null}
                    page="configManagement"
                    pdfId="capture-dashboard"
                />
            </div>
        </div>
    );
};

export default Header;
