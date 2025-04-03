import React from 'react';
import PropTypes from 'prop-types';

import PageHeader from 'Components/PageHeader';

const DashboardLayout = ({ headerText, headerComponents, children, banner }) => {
    return (
        <section className="flex flex-col relative min-h-full">
            {banner}
            <PageHeader classes="z-10 sticky top-0" header={headerText} subHeader="Dashboard">
                <div className="flex flex-1 justify-end h-10">{headerComponents}</div>
            </PageHeader>

            <div
                className="h-full overflow-auto relative p-4 xl:p-6 xxxl:p-8 bg-base-200"
                id="capture-dashboard"
            >
                <div
                    className="grid grid-gap-4 xl:grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense"
                    style={{ '--min-tile-height': '160px' }}
                >
                    {children}
                </div>
            </div>
        </section>
    );
};

DashboardLayout.propTypes = {
    headerText: PropTypes.string.isRequired,
    headerComponents: PropTypes.element.isRequired,
    children: PropTypes.oneOfType([PropTypes.element, PropTypes.arrayOf(PropTypes.element)])
        .isRequired,
    banner: PropTypes.element,
};

export default DashboardLayout;
