import React from 'react';

import Logo from 'Components/icons/logo';
import ThemeToggleButton from 'Components/ThemeToggleButton';
import CLIDownloadButton from 'Components/CLIDownloadButton';
import GlobalSearchButton from 'Components/GlobalSearchButton';
import { useTheme } from 'Containers/ThemeProvider';
import SummaryCounts from './SummaryCounts';
import TopNavBarMenu from './TopNavBarMenu';

const topNavBtnTextClass = 'sm:hidden md:flex uppercase text-sm tracking-wide';
const topNavBtnSvgClass = 'sm:mr-0 md:mr-3 h-4 w-4';
const topNavBtnClass =
    'flex flex-end px-4 no-underline pt-3 pb-2 text-base-600 hover:bg-base-200 items-center cursor-pointer';

const TopNavigation = () => {
    const { isDarkMode } = useTheme();

    return (
        <nav
            className={`top-navigation flex flex-1 justify-between relative bg-header ${
                !isDarkMode ? 'bg-base-200' : 'bg-base-100'
            }`}
            data-testid="top-nav-bar"
        >
            <div className="flex w-full">
                <div
                    className={`flex font-condensed font-600 uppercase py-2 px-4 border-r border-base-400 items-center ${
                        !isDarkMode ? 'bg-base-100' : 'bg-base-0'
                    }`}
                >
                    <Logo className="fill-current text-primary-800" />
                    <div className="pl-1 pt-1 text-sm tracking-wide">Platform</div>
                </div>
                <SummaryCounts />
            </div>
            <div className="flex" data-testid="top-nav-btns">
                <GlobalSearchButton
                    topNavBtnTextClass={topNavBtnTextClass}
                    topNavBtnSvgClass={topNavBtnSvgClass}
                    topNavBtnClass={topNavBtnClass}
                />
                <CLIDownloadButton
                    topNavBtnTextClass={topNavBtnTextClass}
                    topNavBtnSvgClass={topNavBtnSvgClass}
                    topNavBtnClass={topNavBtnClass}
                />
                <ThemeToggleButton />
                <TopNavBarMenu />
            </div>
        </nav>
    );
};

export default TopNavigation;
