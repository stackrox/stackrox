import React from 'react';
import PropTypes from 'prop-types';
import { useTheme } from 'Containers/ThemeProvider';

import WaveBackground from 'images/wave-bg.svg';
import WaveBackground2 from 'images/wave-bg-2.svg';
import PageHeader from 'Components/PageHeader';

const DashboardLayout = ({ headerText, headerComponents, children }) => {
    const { isDarkMode } = useTheme();
    const bgStyle = isDarkMode
        ? {}
        : {
              '--start': 'hsl(240, 100%, 97%)',
              '--end': 'hsl(215, 92%, 95%)'
          };
    return (
        <section className="flex flex-col relative min-h-full">
            <PageHeader
                classes={`z-10 sticky top-0 ${
                    isDarkMode ? 'text-base-600' : 'text-primary-800 bg-gradient-horizontal'
                }`}
                bgStyle={bgStyle}
                header={headerText}
                subHeader="Dashboard"
            >
                <div className="flex flex-1 justify-end h-10">{headerComponents}</div>
            </PageHeader>

            <div
                className={`h-full overflow-auto relative p-4 xl:p-6 xxxl:p-8 ${
                    !isDarkMode ? 'bg-gradient-diagonal ' : ''
                }`}
                style={{ '--start': 'var(--base-200)', '--end': 'var(--primary-200)' }}
                id="capture-dashboard"
            >
                <img
                    className="absolute left-0 pointer-events-none w-full top-0"
                    id="wave-bg"
                    src={WaveBackground}
                    alt="Wave"
                />
                <img
                    className="absolute left-0 pointer-events-none top-0 w-full"
                    id="wave-bg2"
                    src={WaveBackground2}
                    alt="Waves"
                />
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
        .isRequired
};

export default DashboardLayout;
