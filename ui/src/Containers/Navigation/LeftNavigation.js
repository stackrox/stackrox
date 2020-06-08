import React, { useState, useEffect } from 'react';
import * as Icon from 'react-feather';
import { connect } from 'react-redux';
import { withRouter, NavLink } from 'react-router-dom';
import ReactRouterPropTypes from 'react-router-prop-types';
import { createStructuredSelector } from 'reselect';
import find from 'lodash/find';
import PropTypes from 'prop-types';
import { selectors } from 'reducers';

import { useTheme } from 'Containers/ThemeProvider';
import NavigationPanel from './NavigationPanel';
import { apidocsLink, navLinks, productdocsLink } from './LeftSideNavLinks';
import { filterLinksByFeatureFlag } from './navHelpers';

const iconBaseClass = 'h-4 w-4';
const linkBaseClass = 'flex items-center';
const textBaseClass =
    'font-700 font-condensed leading-normal no-underline text-sm tracking-wide uppercase';
const versionBaseClass = 'font-700 leading-normal px-3 py-1 text-xs text-center word-break-all';

const InternalLink = ({
    activeColorClass,
    iconColorClass,
    linkLayoutColorClass,
    navLink,
    showNavigationPanel,
    textColorClass,
}) => (
    <NavLink
        to={navLink.to}
        activeClassName={activeColorClass}
        onClick={showNavigationPanel(navLink)}
        className={`${linkBaseClass} ${linkLayoutColorClass}`}
        data-testid={navLink.data || navLink.text}
    >
        <div className="mr-2">
            <navLink.Icon className={`${iconBaseClass} ${iconColorClass}`} />
        </div>
        <p className={`${textBaseClass} ${textColorClass}`}>{navLink.text}</p>
    </NavLink>
);

const ExternalLink = ({ iconColorClass, linkLayoutColorClass, navLink, textColorClass }) => (
    <a
        href={navLink.to}
        target="_blank"
        rel="noopener noreferrer"
        className={`${linkBaseClass} ${linkLayoutColorClass}`}
    >
        <div className="mr-2">
            <navLink.Icon className={`${iconBaseClass} ${iconColorClass}`} />
        </div>
        <p className={`${textBaseClass} ${textColorClass}`}>{navLink.text}</p>
        <div className="ml-2">
            <Icon.ExternalLink className={`${iconBaseClass} ${iconColorClass}`} />
        </div>
    </a>
);

const versionString = (metadata) => {
    let result = `v${metadata.version}`;
    if (metadata.releaseBuild === false) {
        result += ' [DEV BUILD]';
    }
    return result;
};

const LeftNavigation = ({ featureFlags, location, metadata }) => {
    const { isDarkMode } = useTheme();

    const [panelType, setPanelType] = useState(null);
    const [clickOnPanelItem, setClickOnPanelItem] = useState(false);
    const [selectedPanel, setSelectedPanel] = useState('');

    function closePanel(newClickOnPanelItem, newSelectedPanel) {
        return () => {
            if (newClickOnPanelItem) {
                setClickOnPanelItem(newClickOnPanelItem);
                setSelectedPanel(newSelectedPanel);
            }
            setPanelType(null);
        };
    }

    function showNavigationPanel(navLink) {
        return (e) => {
            if (navLink.panelType && panelType !== navLink.panelType) {
                e.preventDefault();
                setPanelType(navLink.panelType);
            } else {
                if (panelType === navLink.panelType) {
                    e.preventDefault();
                }
                setPanelType(null);
                setClickOnPanelItem(false);
            }
        };
    }

    function renderNavigationPanel() {
        if (!panelType) return '';
        return <NavigationPanel panelType={panelType} onClose={closePanel} />;
    }

    function componentDidMount() {
        window.onpopstate = (e) => {
            const url = e.srcElement.location.pathname;
            const link = find(navLinks, (navLink) => url === navLink.to);
            if (panelType || link) {
                setPanelType(null);
            }
        };
    }

    useEffect(componentDidMount, []);

    const iconColorClass = 'text-primary-400';
    const textColorClass = isDarkMode ? 'text-base-600' : 'text-base-100';
    const versionColorClass = isDarkMode ? 'text-base-500' : 'text-primary-400';

    const menuColorClass = isDarkMode
        ? 'bg-base-100 border-base-300 border-r border-t -mt-px'
        : 'bg-primary-800';

    const linkColorClass = isDarkMode
        ? 'hover:bg-base-200 border-base-400'
        : 'hover:bg-base-700 border-primary-900';

    // Beware: react-router appends the following classes to active/focus link,
    // but they have same specificity as any other Tailwind classes.
    // Therefore, which class wins depends on order of rules in style element,
    // not order of classes in attribute of link element.

    const activeColorClass = isDarkMode
        ? 'bg-primary-200 hover:bg-primary-200'
        : 'bg-primary-700 hover:bg-primary-700';

    const focusColorClass = isDarkMode
        ? 'bg-primary-300 hover:bg-primary-300'
        : 'bg-base-800 hover:bg-base-800';

    function getActiveColorClass(navLink) {
        const { pathname } = location;

        if (navLink.to !== '') {
            return activeColorClass;
        }

        if (navLink.paths && navLink.paths.some((path) => pathname.includes(path))) {
            return activeColorClass;
        }

        if (panelType && panelType === navLink.panelType) {
            return focusColorClass;
        }

        if (!panelType && clickOnPanelItem && selectedPanel === navLink.text.toLowerCase()) {
            return focusColorClass;
        }

        return '';
    }

    // API Reference is not in navLinks array because:
    // any extra vertical space is above it;
    // therefore, it has border top;
    // it has about half the height of the links above it.

    return (
        <>
            <div
                className={`flex flex-col flex-none h-full overflow-auto z-60 ignore-react-onclickoutside ${menuColorClass}`}
            >
                <nav className="flex flex-col flex-grow left-navigation">
                    <ul className="flex flex-col h-full">
                        {filterLinksByFeatureFlag(featureFlags, navLinks).map((navLink) => (
                            <li key={navLink.text}>
                                <InternalLink
                                    activeColorClass={getActiveColorClass(navLink)}
                                    iconColorClass={iconColorClass}
                                    linkLayoutColorClass={`border-b h-18 px-3 ${linkColorClass}`}
                                    navLink={navLink}
                                    showNavigationPanel={showNavigationPanel}
                                    textColorClass={textColorClass}
                                />
                            </li>
                        ))}
                        <li className="flex flex-col flex-grow justify-end">
                            <InternalLink
                                activeColorClass={getActiveColorClass(apidocsLink)}
                                iconColorClass={iconColorClass}
                                linkLayoutColorClass={`border-b border-t h-8 px-3 ${linkColorClass}`}
                                navLink={apidocsLink}
                                showNavigationPanel={showNavigationPanel}
                                textColorClass={textColorClass}
                            />
                        </li>
                        <li>
                            <ExternalLink
                                iconColorClass={iconColorClass}
                                linkLayoutColorClass={`border-b h-8 pl-3 pr-2 ${linkColorClass}`}
                                navLink={productdocsLink}
                                textColorClass={textColorClass}
                            />
                        </li>
                    </ul>
                </nav>
                <p className={`left-navigation ${versionBaseClass} ${versionColorClass}`}>
                    {versionString(metadata)}
                </p>
            </div>
            {renderNavigationPanel()}
        </>
    );
};

LeftNavigation.propTypes = {
    location: ReactRouterPropTypes.location.isRequired,
    featureFlags: PropTypes.arrayOf(
        PropTypes.shape({
            envVar: PropTypes.string.isRequired,
            enabled: PropTypes.bool.isRequired,
        })
    ).isRequired,
    metadata: PropTypes.shape({
        version: PropTypes.string,
        releaseBuild: PropTypes.bool,
        licenseStatus: PropTypes.string,
    }),
};

LeftNavigation.defaultProps = {
    metadata: {
        version: 'latest',
        releaseBuild: false,
    },
};

const mapStateToProps = createStructuredSelector({
    featureFlags: selectors.getFeatureFlags,
    metadata: selectors.getMetadata,
});

export default withRouter(connect(mapStateToProps)(LeftNavigation));
