import React from 'react';

import ExportButton from 'Components/ExportButton';
import useCaseTypes from 'constants/useCaseTypes';
import PoliciesTile from './PoliciesTile';
import CISControlsTile from './CISControlsTile';
import AppMenu from './AppMenu';
import RBACMenu from './RBACMenu';

const Header = () => {
    return (
        <div className="flex flex-1 justify-end">
            <div className="border-base-400 border-r-2 mr-1 flex ">
                <PoliciesTile />
                <CISControlsTile />
                <div className="flex w-32 mr-2">
                    <AppMenu />
                </div>
                <div className="flex w-32 mr-3 ">
                    <RBACMenu />
                </div>
            </div>
            <div className="flex items-center self-center">
                <ExportButton
                    fileName="Config Management Dashboard Report"
                    type={null}
                    page={useCaseTypes.CONFIG_MANAGEMENT}
                    pdfId="capture-dashboard"
                />
            </div>
        </div>
    );
};

export default Header;
