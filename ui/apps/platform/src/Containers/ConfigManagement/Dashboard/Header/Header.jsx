import React from 'react';
import PropTypes from 'prop-types';
import ExportButton from 'Components/ExportButton';
import useCaseTypes from 'constants/useCaseTypes';
import usePermissions from 'hooks/usePermissions';
import PoliciesTile from './PoliciesTile';
import CISControlsTile from './CISControlsTile';
import AppMenu from './AppMenu';
import RBACMenu from './RBACMenu';

const Header = ({ isExporting, setIsExporting }) => {
    const { hasReadAccess } = usePermissions();
    const hasReadAccessForPoliciesTile = hasReadAccess('WorkflowAdministration');
    const hasReadAccessForCISControlsTile = hasReadAccess('Compliance');
    return (
        <div className="flex flex-1 justify-end">
            <div className="border-base-400 border-r-2 mr-1 flex ">
                {hasReadAccessForPoliciesTile && <PoliciesTile />}
                {hasReadAccessForCISControlsTile && <CISControlsTile />}
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
                    isExporting={isExporting}
                    setIsExporting={setIsExporting}
                />
            </div>
        </div>
    );
};

Header.propTypes = {
    isExporting: PropTypes.bool.isRequired,
    setIsExporting: PropTypes.func.isRequired,
};

export default Header;
