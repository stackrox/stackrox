import React from 'react';
import PropTypes from 'prop-types';
import { useTheme } from 'Containers/ThemeProvider';

import PageHeader from 'Components/PageHeader';
import ScanButton from 'Containers/Compliance/ScanButton';
import ExportButton from 'Components/ExportButton';
import entityTypes from 'constants/entityTypes';
import useCaseTypes from 'constants/useCaseTypes';
import Tile from './Tile';

const ComplianceDashboardHeader = ({ classes, bgStyle, isExporting, setIsExporting }) => {
    const { isDarkMode } = useTheme();
    const darkModeClasses = `${
        isDarkMode ? 'text-base-600 hover:bg-primary-200' : 'text-base-100 hover:bg-primary-800'
    }`;

    return (
        <PageHeader classes={classes} bgStyle={bgStyle} header="Compliance" subHeader="Dashboard">
            <div className="flex w-full justify-end">
                <div className="flex">
                    <Tile entityType={entityTypes.CLUSTER} position="first" />
                    <Tile entityType={entityTypes.NAMESPACE} position="middle" />
                    <Tile entityType={entityTypes.NODE} position="middle" />
                    <Tile entityType={entityTypes.DEPLOYMENT} position="last" />
                    <div className="ml-1 border-l-2 border-base-400 mr-3" />
                    <div className="flex items-center">
                        <div className="flex items-center">
                            <ScanButton
                                className={`flex items-center justify-center border-2 btn btn-base h-10 uppercase lg:min-w-32 xl:min-w-43 ${darkModeClasses}`}
                                text="Scan environment"
                                textClass="hidden lg:block"
                                textCondensed="Scan all"
                                clusterId="*"
                                standardId="*"
                            />
                        </div>
                        <div className="flex items-center">
                            <ExportButton
                                fileName="Compliance Dashboard Report"
                                textClass="hidden lg:block"
                                type="ALL"
                                page={useCaseTypes.COMPLIANCE}
                                pdfId="capture-dashboard"
                                isExporting={isExporting}
                                setIsExporting={setIsExporting}
                            />
                        </div>
                    </div>
                </div>
            </div>
        </PageHeader>
    );
};

ComplianceDashboardHeader.propTypes = {
    classes: PropTypes.string,
    bgStyle: PropTypes.shape({}),
    isExporting: PropTypes.bool.isRequired,
    setIsExporting: PropTypes.func.isRequired,
};

ComplianceDashboardHeader.defaultProps = {
    classes: null,
    bgStyle: null,
};

export default ComplianceDashboardHeader;
