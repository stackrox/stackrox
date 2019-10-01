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
              boxShadow: 'hsla(230, 75%, 63%, 0.62) 0px 5px 30px 0px',
              '--start': 'hsl(226, 70%, 60%)',
              '--end': 'hsl(226, 64%, 56%)'
          };

    return (
        <section className="flex flex-col relative min-h-full">
            <PageHeader
                classes={`bg-gradient-horizontal z-10 sticky pin-t ${
                    isDarkMode ? 'text-base-600' : 'text-base-100'
                }`}
                bgStyle={bgStyle}
                header={headerText}
                subHeader="Dashboard"
            >
                <div className="flex flex-1 justify-end">{headerComponents}</div>
            </PageHeader>
            <img
                className="absolute pin-l pointer-events-none z-10 w-full"
                id="wave-bg2"
                src={WaveBackground2}
                style={{ mixBlendMode: 'lighten', top: '-60px' }}
                alt="Waves"
            />
            <div
                className="relative bg-gradient-diagonal p-6 xxxl:p-8"
                style={{ '--start': 'var(--base-200)', '--end': 'var(--primary-100)' }}
                id="capture-dashboard"
            >
                <img
                    className="absolute pin-l pointer-events-none w-full"
                    id="wave-bg"
                    src={WaveBackground}
                    style={{ top: '-130px' }}
                    alt="Wave"
                />
                <div
                    className="grid grid-gap-6 xxxl:grid-gap-8 md:grid-auto-fit xxl:grid-auto-fit-wide md:grid-dense"
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
    children: PropTypes.element.isRequired
};

export default DashboardLayout;
